/**
 * 认证钩子示例脚本 - 在请求发送前处理认证信息
 * 
 * 这个脚本会在每个请求前执行，向请求中添加时间戳和签名
 * 要求请求体必须是JSON格式
 */

// 这个函数会被RenderAPI自动调用，用于处理请求
function processRequest(request) {
    console.log("正在处理认证请求...");
    
    // 确保请求包含请求体
    if (!request.body) {
        // 如果没有请求体，则创建一个空对象
        request.body = {};
    }
    
    // 获取当前时间戳（毫秒）
    const timestamp = new Date().getTime();
    
    // 添加时间戳到请求体
    request.body.timestamp = timestamp;
    
    // 计算请求签名（这里使用一个简单的方法，实际应用中可以使用更复杂的加密）
    const apiSecret = "YOUR_SECRET_KEY"; // 实际中应该从安全存储获取
    
    // 创建要签名的字符串
    const dataToSign = JSON.stringify(request.body) + apiSecret;
    
    // 计算简单签名（此处仅为示例，实际应用中应使用加密函数如MD5、SHA256等）
    let signature = simpleHash(dataToSign);
    
    // 添加签名到请求体
    request.body.signature = signature;
    
    console.log("已添加认证信息：", "timestamp=" + timestamp, "signature=" + signature);
    
    // 返回修改后的请求
    return request;
}

// 简单的哈希函数示例（非加密用途）
function simpleHash(str) {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
        const char = str.charCodeAt(i);
        hash = ((hash << 5) - hash) + char;
        hash = hash & hash; // Convert to 32bit integer
    }
    return hash.toString(16); // 转为16进制字符串
}

// 你也可以添加其他辅助函数
function formatDate(date) {
    return date.toISOString();
}

// 测试代码（这部分不会被RenderAPI执行）
if (typeof window !== 'undefined') {
    // 仅在浏览器环境中测试
    const testRequest = {
        body: {
            user: "testUser",
            action: "login"
        }
    };
    
    console.log("测试结果:", processRequest(testRequest));
} 