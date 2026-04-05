// 用 Node.js 生成 icon.png（只需运行一次）
// node make_icon.js
const { createCanvas } = require('canvas');
const fs = require('fs');

const size = 128;
const canvas = createCanvas(size, size);
const ctx = canvas.getContext('2d');

ctx.fillStyle = '#2563eb';
ctx.beginPath();
ctx.roundRect(0, 0, size, size, 24);
ctx.fill();

ctx.fillStyle = '#fff';
ctx.font = 'bold 72px serif';
ctx.textAlign = 'center';
ctx.textBaseline = 'middle';
ctx.fillText('I', size / 2, size / 2);

fs.writeFileSync('icon.png', canvas.toBuffer('image/png'));
console.log('icon.png generated');
