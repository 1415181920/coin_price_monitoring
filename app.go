package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Config 配置结构
type Config struct {
	Proxy struct {
		Enabled bool   `json:"enabled"`
		Address string `json:"address"`
	} `json:"proxy"`
	UpdateInterval int `json:"update_interval"`
}

// App struct
type App struct {
	ctx        context.Context
	btcPrice   string
	ethPrice   string
	lastUpdate time.Time
	config     *Config
}

// BinancePrice 币安API响应结构
type BinancePrice struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
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
		UpdateInterval: 10, // 默认10秒
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
		btcPrice: "加载中...",
		ethPrice: "加载中...",
		config:   loadConfig(),
	}
}

// getHTTPClient 创建支持代理的HTTP客户端
func (a *App) getHTTPClient() *http.Client {
	transport := &http.Transport{}

	// 代理优先级：
	// 1. 环境变量 HTTP_PROXY / HTTPS_PROXY
	// 2. 配置文件中的代理设置
	// 3. 不使用代理

	if proxyEnv := os.Getenv("HTTP_PROXY"); proxyEnv != "" {
		// 优先使用环境变量
		if proxyURL, err := url.Parse(proxyEnv); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
			fmt.Printf("✓ 使用环境变量代理: %s\n", proxyEnv)
		}
	} else if proxyEnv := os.Getenv("HTTPS_PROXY"); proxyEnv != "" {
		if proxyURL, err := url.Parse(proxyEnv); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
			fmt.Printf("✓ 使用环境变量代理: %s\n", proxyEnv)
		}
	} else if a.config != nil && a.config.Proxy.Enabled && a.config.Proxy.Address != "" {
		// 使用配置文件中的代理
		if proxyURL, err := url.Parse(a.config.Proxy.Address); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
			fmt.Printf("✓ 使用配置代理: %s\n", a.config.Proxy.Address)
		} else {
			fmt.Printf("✗ 代理地址解析失败: %s\n", a.config.Proxy.Address)
		}
	} else {
		fmt.Println("⚠ 未配置代理，直连币安API（国内可能无法访问）")
	}

	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	
	// 启动时立即获取一次价格
	go a.fetchPrices()
	// 启动定时器，每10秒更新一次价格
	go a.startPriceUpdater()
}

// GetPrices 获取当前价格（供前端调用）
func (a *App) GetPrices() PriceData {
	return PriceData{
		BTC:        a.btcPrice,
		ETH:        a.ethPrice,
		LastUpdate: a.lastUpdate.Format("15:04:05"),
	}
}

// fetchPrice 从币安API获取指定币种价格
func (a *App) fetchPrice(symbol string) (string, error) {
	// 使用支持代理的HTTP客户端
	client := a.getHTTPClient()

	apiURL := fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", symbol)
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取失败: %w", err)
	}

	var priceData BinancePrice
	if err := json.Unmarshal(body, &priceData); err != nil {
		return "", fmt.Errorf("解析失败: %w", err)
	}

	// 格式化价格
	price := priceData.Price
	if len(price) > 3 {
		// 截取整数部分和2位小数
		dotIndex := -1
		for i, c := range price {
			if c == '.' {
				dotIndex = i
				break
			}
		}
		if dotIndex != -1 && dotIndex+3 < len(price) {
			price = price[:dotIndex+3]
		}
	}

	return price, nil
}

// fetchPrices 获取所有价格
func (a *App) fetchPrices() {
	// 获取BTC价格
	btcPrice, err := a.fetchPrice("BTCUSDT")
	if err != nil {
		fmt.Printf("获取BTC价格失败: %v\n", err)
		a.btcPrice = "N/A"
	} else {
		a.btcPrice = btcPrice
		fmt.Printf("BTC价格: $%s\n", btcPrice)
	}

	// 获取ETH价格
	ethPrice, err := a.fetchPrice("ETHUSDT")
	if err != nil {
		fmt.Printf("获取ETH价格失败: %v\n", err)
		a.ethPrice = "N/A"
	} else {
		a.ethPrice = ethPrice
		fmt.Printf("ETH价格: $%s\n", ethPrice)
	}

	a.lastUpdate = time.Now()

	// 发送事件到前端更新UI
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "price-update", a.GetPrices())
		
		// 更新窗口标题
		title := fmt.Sprintf("BTC: $%s | ETH: $%s", a.btcPrice, a.ethPrice)
		runtime.WindowSetTitle(a.ctx, title)
	}
}

// startPriceUpdater 启动价格更新定时器
func (a *App) startPriceUpdater() {
	interval := time.Duration(a.config.UpdateInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("✓ 价格更新间隔: %d秒\n", a.config.UpdateInterval)

	for {
		select {
		case <-ticker.C:
			a.fetchPrices()
		case <-a.ctx.Done():
			return
		}
	}
}
