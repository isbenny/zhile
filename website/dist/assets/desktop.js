(() => {
  const button = document.getElementById('copy-install');
  const prompt = document.getElementById('install-prompt');
  const status = document.getElementById('copy-status');
  if (!button || !prompt || !status) return;
  button.addEventListener('click', async () => {
    try {
      await navigator.clipboard.writeText(prompt.value);
      button.textContent = '已复制 ✓';
      status.textContent = '已复制。现在粘贴到你自己电脑上的 AI 助手中。';
    } catch (_) {
      prompt.focus();
      prompt.select();
      status.textContent = '已选中提示词。请按 Ctrl+C / Command+C，或长按文字复制。';
    }
  });
})();
