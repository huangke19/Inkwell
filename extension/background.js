chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: 'inkwell-add',
    title: '加入 Inkwell：「%s」',
    contexts: ['selection'],
  });
});

chrome.contextMenus.onClicked.addListener((info, tab) => {
  if (info.menuItemId !== 'inkwell-add') return;
  const text = info.selectionText.trim();
  if (!text) return;

  chrome.tabs.sendMessage(tab.id, { type: 'inkwell-add', text });
});
