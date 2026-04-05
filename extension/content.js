(() => {
  const INKWELL_URL = 'http://localhost:9090/words';
  let bubble = null;
  let hideTimer = null;

  function getContext() {
    // 尝试抓取 Twitter 推文正文作为上下文
    const sel = window.getSelection();
    if (!sel || sel.rangeCount === 0) return '';

    // 从选中位置向上找推文容器
    let node = sel.getRangeAt(0).commonAncestorContainer;
    while (node && node !== document.body) {
      // Twitter 推文文本容器的 data-testid
      if (node.dataset && (
        node.dataset.testid === 'tweetText' ||
        node.dataset.testid === 'tweet'
      )) {
        return node.innerText || node.textContent || '';
      }
      node = node.parentElement;
    }
    // 兜底：取选中词前后各 100 个字符
    const range = sel.getRangeAt(0);
    const container = range.commonAncestorContainer.textContent || '';
    const start = Math.max(0, range.startOffset - 100);
    const end = Math.min(container.length, range.endOffset + 100);
    return container.slice(start, end).trim();
  }

  function createBubble() {
    if (bubble) return bubble;
    bubble = document.createElement('div');
    bubble.id = 'inkwell-bubble';
    document.body.appendChild(bubble);

    bubble.addEventListener('mousedown', e => e.stopPropagation());
    return bubble;
  }

  function showBubble(word, x, y) {
    clearTimeout(hideTimer);
    const b = createBubble();
    b.className = '';
    b.innerHTML = `<span class="inkwell-icon">✏️</span> 加入 Inkwell：<strong>${word}</strong>`;
    b.style.left = x + 'px';
    b.style.top = (y - 44) + 'px';

    b.onclick = () => addWord(word);
  }

  function hideBubble() {
    if (bubble) {
      bubble.remove();
      bubble = null;
    }
  }

  async function addWord(word) {
    const context = getContext();
    const b = createBubble();
    b.innerHTML = `<span class="inkwell-icon">⏳</span> 添加中…`;
    b.onclick = null;

    try {
      const res = await fetch(INKWELL_URL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ word, context }),
      });

      if (res.status === 201) {
        b.className = 'success';
        b.innerHTML = `<span class="inkwell-icon">✓</span> 已添加：<strong>${word}</strong>`;
      } else if (res.status === 409) {
        b.className = 'duplicate';
        b.innerHTML = `<span class="inkwell-icon">·</span> 已在词库中`;
      } else {
        b.className = 'error';
        b.innerHTML = `<span class="inkwell-icon">✕</span> 添加失败`;
      }
    } catch {
      b.className = 'error';
      b.innerHTML = `<span class="inkwell-icon">✕</span> 无法连接 Inkwell`;
    }

    hideTimer = setTimeout(hideBubble, 2000);
  }

  document.addEventListener('mouseup', e => {
    // 点在气泡上不处理
    if (bubble && bubble.contains(e.target)) return;

    const sel = window.getSelection();
    const word = sel ? sel.toString().trim() : '';

    // 只处理单个英文单词（允许连字符，如 well-known）
    if (/^[a-zA-Z][a-zA-Z'-]*[a-zA-Z]$/.test(word) || /^[a-zA-Z]$/.test(word)) {
      showBubble(word.toLowerCase(), e.pageX, e.pageY);
    } else {
      hideBubble();
    }
  });

  document.addEventListener('mousedown', e => {
    if (bubble && !bubble.contains(e.target)) {
      hideBubble();
    }
  });

  document.addEventListener('keydown', e => {
    if (e.key === 'Escape') hideBubble();
  });
})();
