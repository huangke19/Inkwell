fetch('http://localhost:9090/')
  .then(() => {
    document.getElementById('dot').className = 'dot on';
    document.getElementById('status-text').textContent = 'Inkwell 运行中';
  })
  .catch(() => {
    document.getElementById('status-text').textContent = '未检测到 Inkwell 服务';
  });
