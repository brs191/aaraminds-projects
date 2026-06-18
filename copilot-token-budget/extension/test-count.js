const reader = require('./out/session/reader');

(async () => {
  try {
    const all = await reader.readSessions();
    const thisMonth = await reader.readThisMonth();
    const ideMarkeds = all.filter(s => s.hasIdeActivity);
    
    console.log(`\n=== Session Summary ===`);
    console.log(`Total readable sessions: ${all.length}`);
    console.log(`Marked with IDE activity: ${ideMarkeds.length}`);
    console.log(`CLI-only: ${all.length - ideMarkeds.length}`);
    console.log(`\nThis month: ${thisMonth.length}`);
    
    console.log(`\n=== Monthly Breakdown ===`);
    const byMonth = new Map();
    for (const s of all) {
      const d = new Date(s.startTime || s.endTime || new Date());
      const m = d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0');
      byMonth.set(m, (byMonth.get(m) || 0) + 1);
    }
    for (const [month, count] of [...byMonth.entries()].sort()) {
      console.log(`${month}: ${count} sessions`);
    }
  } catch (err) {
    console.error("Error:", err.message);
  }
})();
