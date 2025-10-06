<script setup>
import { ref, onMounted } from 'vue'
import { GetPrices } from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'

const btcPrice = ref('...')
const ethPrice = ref('...')

// 获取价格
const fetchPrices = async () => {
  try {
    const result = await GetPrices()
    btcPrice.value = result.btc
    ethPrice.value = result.eth
  } catch (error) {
    console.error('获取价格失败:', error)
  }
}

onMounted(() => {
  // 初始加载
  fetchPrices()
  
  // 监听价格更新事件
  EventsOn('price-update', (data) => {
    btcPrice.value = data.btc
    ethPrice.value = data.eth
  })
})
</script>

<template>
  <div class="container">
    <div class="price-bar">
      <div class="price-item btc">
        <span class="value">${{ btcPrice }}</span>
      </div>
      <div class="price-item eth">
        <span class="value">${{ ethPrice }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

.container {
  width: 100vw;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #000;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Helvetica', 'Arial', sans-serif;
  overflow: hidden;
  --wails-draggable: drag;
}

.price-bar {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 2px;
  background: #000;
  border-radius: 4px;
  cursor: move;
  user-select: none;
  -webkit-user-select: none;
}

.price-item {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.label {
  font-size: 12px;
  font-weight: 600;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  min-width: 32px;
}

.value {
  font-size: 14px;
  font-weight: 700;
  font-family: 'Courier New', 'Consolas', monospace;
  color: #10b981;
  text-shadow: 0 0 10px rgba(16, 185, 129, 0.3);
  flex: 1;
}

.btc .label {
  color: #f59e0b;
}

.btc .value {
  color: #fbbf24;
}

.eth .label {
  color: #8b5cf6;
}

.eth .value {
  color: #a78bfa;
}

/* 鼠标悬停效果 */
.price-bar:hover {
  background: rgba(0, 0, 0, 0.9);
  border-color: rgba(255, 255, 255, 0.25);
  transition: all 0.2s ease;
}

/* 禁用滚动条并确保透明 */
html, body {
  overflow: hidden;
  margin: 0;
  padding: 0;
  background: transparent !important;
}

#app {
  background: transparent !important;
}
</style>
