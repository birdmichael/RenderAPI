// 请求前处理脚本
function processRequest(request) {
  console.log("在JavaScript中处理请求...");
  
  // 添加时间戳字段
  request.body.timestamp = Date.now();
  
  // 添加自定义字段
  request.body.processed_by = "script";
  
  return request;
} 