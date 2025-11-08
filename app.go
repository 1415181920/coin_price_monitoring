package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Config 配置结构
type Config struct {
	Proxy struct {
		Enabled bool   `json:"enabled"`
		Address string `json:"address"`
	} `json:"proxy"`
	UpdateInterval int `json:"update_interval"` // 已弃用，保留用于兼容性
}

// App struct
type App struct {
	ctx        context.Context
	btcPrice   string
	ethPrice   string
	lastUpdate time.Time
	config     *Config
	wsConn     *websocket.Conn
	mu         sync.RWMutex // 保护价格数据
	reconnect  chan bool    // 重连信号
}

// BitgetTickerData Bitget Ticker数据结构
type BitgetTickerData struct {
	InstId    string `json:"instId"`    // 交易对ID
	LastPr    string `json:"lastPr"`    // 最新价格
	BidPr     string `json:"bidPr"`     // 最优买价
	AskPr     string `json:"askPr"`     // 最优卖价
	High24h   string `json:"high24h"`   // 24小时最高价
	Low24h    string `json:"low24h"`    // 24小时最低价
	Change24h string `json:"change24h"` // 24小时涨跌幅
}

// BitgetWSMessage WebSocket消息结构
type BitgetWSMessage struct {
	Op     string               `json:"op,omitempty"`     // 操作类型：subscribe/ping/pong
	Event  string               `json:"event,omitempty"`  // 事件类型：subscribe
	Args   []BitgetSubscribeArg `json:"args,omitempty"`   // 订阅参数
	Action string               `json:"action,omitempty"` // 动作：snapshot/update
	Arg    *BitgetSubscribeArg  `json:"arg,omitempty"`    // 推送的订阅信息
	Data   []BitgetTickerData   `json:"data,omitempty"`   // Ticker数据
	Code   string               `json:"code,omitempty"`   // 响应代码
	Msg    string               `json:"msg,omitempty"`    // 响应消息
}

// BitgetSubscribeArg 订阅参数
type BitgetSubscribeArg struct {
	InstType string `json:"instType"` // 产品类型：USDT-FUTURES
	Channel  string `json:"channel"`  // 频道：ticker
	InstId   string `json:"instId"`   // 交易对：BTCUSDT
}

// PriceData 价格数据结构
type PriceData struct {
	BTC        string `json:"btc"`
	ETH        string `json:"eth"`
	LastUpdate string `json:"lastUpdate"`
}

// loadConfig 加载配置文件
func loadConfig() *Config {
	config := &Config{
		UpdateInterval: 10, // 保留用于兼容性，WebSocket为实时推送
	}
	config.Proxy.Enabled = true
	config.Proxy.Address = "http://127.0.0.1:7897" // 默认Clash端口

	// 尝试读取配置文件
	data, err := os.ReadFile("config.json")
	if err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			fmt.Printf("配置文件解析失败，使用默认配置: %v\n", err)
		} else {
			fmt.Println("已加载配置文件 config.json")
		}
	} else {
		fmt.Println("未找到配置文件，使用默认配置（代理: http://127.0.0.1:7897）")
	}

	return config
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		btcPrice:  "加载中...",
		ethPrice:  "加载中...",
		config:    loadConfig(),
		reconnect: make(chan bool, 1),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	fmt.Println("启动 Bitget WebSocket 连接...")
	// 启动WebSocket连接
	go a.startWebSocket()
}

// GetPrices 获取当前价格（供前端调用）
func (a *App) GetPrices() PriceData {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return PriceData{
		BTC:        a.btcPrice,
		ETH:        a.ethPrice,
		LastUpdate: a.lastUpdate.Format("15:04:05"),
	}
}

// startWebSocket 启动WebSocket连接（包含重连逻辑）
func (a *App) startWebSocket() {
	retryDelay := 1 * time.Second
	maxRetryDelay := 30 * time.Second

	for {
		err := a.connectAndListen()
		if err != nil {
			fmt.Printf("WebSocket连接错误: %v\n", err)
		}

		// 检查是否需要退出
		select {
		case <-a.ctx.Done():
			fmt.Println("应用退出，关闭WebSocket连接")
			return
		default:
		}

		// 指数退避重连
		fmt.Printf("将在 %v 后重连...\n", retryDelay)
		time.Sleep(retryDelay)

		retryDelay *= 2
		if retryDelay > maxRetryDelay {
			retryDelay = maxRetryDelay
		}
	}
}

// connectAndListen 连接WebSocket并监听消息
func (a *App) connectAndListen() error {
	// 创建WebSocket拨号器
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	// 配置代理
	if a.config != nil && a.config.Proxy.Enabled && a.config.Proxy.Address != "" {
		proxyURL, err := url.Parse(a.config.Proxy.Address)
		if err != nil {
			fmt.Printf("✗ 代理地址解析失败: %s\n", a.config.Proxy.Address)
		} else {
			dialer.Proxy = http.ProxyURL(proxyURL)
			fmt.Printf("✓ 使用代理连接WebSocket: %s\n", a.config.Proxy.Address)
		}
	} else if proxyEnv := os.Getenv("HTTP_PROXY"); proxyEnv != "" {
		if proxyURL, err := url.Parse(proxyEnv); err == nil {
			dialer.Proxy = http.ProxyURL(proxyURL)
			fmt.Printf("✓ 使用环境变量代理: %s\n", proxyEnv)
		}
	} else if proxyEnv := os.Getenv("HTTPS_PROXY"); proxyEnv != "" {
		if proxyURL, err := url.Parse(proxyEnv); err == nil {
			dialer.Proxy = http.ProxyURL(proxyURL)
			fmt.Printf("✓ 使用环境变量代理: %s\n", proxyEnv)
		}
	} else {
		fmt.Println("⚠ 未配置代理，直连Bitget API（国内可能需要代理）")
	}

	// 连接到Bitget WebSocket
	wsURL := "wss://ws.bitget.com/v2/ws/public"
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("连接WebSocket失败: %w", err)
	}

	a.mu.Lock()
	a.wsConn = conn
	a.mu.Unlock()

	fmt.Printf("✓ 已连接到 Bitget WebSocket: %s\n", wsURL)

	// 订阅ticker频道
	if err := a.subscribeToTickers(); err != nil {
		conn.Close()
		return fmt.Errorf("订阅失败: %w", err)
	}

	// 启动心跳
	go a.startHeartbeat()

	// 监听消息
	return a.handleWebSocketMessages()
}

// subscribeToTickers 订阅BTC和ETH的ticker数据
func (a *App) subscribeToTickers() error {
	subscribeMsg := BitgetWSMessage{
		Op: "subscribe",
		Args: []BitgetSubscribeArg{
			{
				InstType: "USDT-FUTURES",
				Channel:  "ticker",
				InstId:   "BTCUSDT",
			},
			{
				InstType: "USDT-FUTURES",
				Channel:  "ticker",
				InstId:   "ETHUSDT",
			},
		},
	}

	a.mu.RLock()
	conn := a.wsConn
	a.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("WebSocket连接未建立")
	}

	err := conn.WriteJSON(subscribeMsg)
	if err != nil {
		return fmt.Errorf("发送订阅消息失败: %w", err)
	}

	fmt.Println("✓ 已订阅 BTCUSDT 和 ETHUSDT ticker频道")
	return nil
}

// startHeartbeat 启动心跳维持连接
func (a *App) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.mu.RLock()
			conn := a.wsConn
			a.mu.RUnlock()

			if conn == nil {
				return
			}

			pingMsg := map[string]string{"op": "ping"}
			err := conn.WriteJSON(pingMsg)
			if err != nil {
				fmt.Printf("发送心跳失败: %v\n", err)
				return
			}

		case <-a.ctx.Done():
			return
		}
	}
}

// handleWebSocketMessages 处理WebSocket消息
func (a *App) handleWebSocketMessages() error {
	a.mu.RLock()
	conn := a.wsConn
	a.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("WebSocket连接未建立")
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("读取消息失败: %w", err)
		}

		var wsMsg BitgetWSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			fmt.Printf("解析消息失败: %v\n", err)
			continue
		}

		// 处理不同类型的消息
		switch {
		case wsMsg.Op == "pong":
			// 心跳响应，静默处理
			continue

		case wsMsg.Event == "subscribe":
			// 订阅响应
			if wsMsg.Arg != nil {
				fmt.Printf("✓ 订阅成功: %s %s\n", wsMsg.Arg.InstId, wsMsg.Arg.Channel)
			}

		case wsMsg.Code != "":
			// 错误响应
			if wsMsg.Code == "0" {
				fmt.Printf("✓ 操作成功\n")
			} else {
				fmt.Printf("✗ 操作失败: %s - %s\n", wsMsg.Code, wsMsg.Msg)
			}

		case wsMsg.Action == "snapshot" || wsMsg.Action == "update":
			// Ticker数据推送
			a.handleTickerData(&wsMsg)
		}
	}
}

// handleTickerData 处理ticker数据更新
func (a *App) handleTickerData(wsMsg *BitgetWSMessage) {
	if len(wsMsg.Data) == 0 {
		return
	}

	for _, ticker := range wsMsg.Data {
		price := a.formatPrice(ticker.LastPr)

		a.mu.Lock()
		switch ticker.InstId {
		case "BTCUSDT":
			a.btcPrice = price
			fmt.Printf("BTC价格更新: $%s\n", price)
		case "ETHUSDT":
			a.ethPrice = price
			fmt.Printf("ETH价格更新: $%s\n", price)
		}
		a.lastUpdate = time.Now()
		a.mu.Unlock()
	}

	// 发送事件到前端更新UI
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "price-update", a.GetPrices())

		// 更新窗口标题
		a.mu.RLock()
		title := fmt.Sprintf("BTC: $%s | ETH: $%s", a.btcPrice, a.ethPrice)
		a.mu.RUnlock()
		runtime.WindowSetTitle(a.ctx, title)
	}
}

// formatPrice 格式化价格（保留2位小数）
func (a *App) formatPrice(price string) string {
	if price == "" {
		return "N/A"
	}

	// 截取整数部分和2位小数
	dotIndex := -1
	for i, c := range price {
		if c == '.' {
			dotIndex = i
			break
		}
	}

	if dotIndex != -1 && dotIndex+3 < len(price) {
		return price[:dotIndex+3]
	}

	return price
}
