// Animation: planets orbiting a sun. Anything that moves uses t (seconds).
function draw(ctx, w, h, t) {
  const cx = w / 2, cy = h / 2;
  ctx.fillStyle = '#F2B134';
  ctx.beginPath(); ctx.arc(cx, cy, 26, 0, Math.PI * 2); ctx.fill();
  const planets = [{ r: 70, size: 6, speed: 1.6 }, { r: 115, size: 10, speed: 1 }, { r: 165, size: 8, speed: 0.6 }];
  planets.forEach((p, i) => {
    ctx.strokeStyle = nox.colors.grid;
    ctx.beginPath(); ctx.arc(cx, cy, p.r, 0, Math.PI * 2); ctx.stroke();
    const a = t * p.speed + i * 2;
    ctx.fillStyle = nox.colors.palette[i + 1];
    ctx.beginPath(); ctx.arc(cx + Math.cos(a) * p.r, cy + Math.sin(a) * p.r, p.size, 0, Math.PI * 2); ctx.fill();
  });
  nox.text(ctx, 'Sluneční soustava', 20, 32, { size: 16, bold: true });
}
