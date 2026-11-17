// Diagram: boxes joined by arrows, drawn once (no draw function needed,
// but draw() keeps it sharp when the window is resized).
function draw(ctx, w, h) {
  const steps = ['Nápad', 'Návrh', 'Kód', 'Test', 'Vydání'];
  const bw = Math.min(130, (w - 60) / steps.length - 24), bh = 54, y = h / 2 - bh / 2;
  const gap = (w - steps.length * bw) / (steps.length + 1);
  steps.forEach((s, i) => {
    const x = gap + i * (bw + gap);
    ctx.fillStyle = nox.colors.palette[i % nox.colors.palette.length];
    ctx.beginPath(); ctx.roundRect(x, y, bw, bh, 12); ctx.fill();
    nox.text(ctx, s, x + bw / 2, y + bh / 2 + 5, { size: 14, bold: true, color: '#fff', align: 'center' });
    if (i < steps.length - 1) { // arrow to the next box
      const ax = x + bw + 6, bx = x + bw + gap - 6, ay = y + bh / 2;
      ctx.strokeStyle = ctx.fillStyle = nox.colors.dim; ctx.lineWidth = 2;
      ctx.beginPath(); ctx.moveTo(ax, ay); ctx.lineTo(bx, ay); ctx.stroke();
      ctx.beginPath(); ctx.moveTo(bx, ay); ctx.lineTo(bx - 8, ay - 5); ctx.lineTo(bx - 8, ay + 5); ctx.fill();
    }
  });
  nox.text(ctx, 'Jak vzniká software', 20, 32, { size: 16, bold: true });
}
