const reader = require('./out/session/reader');

(async () => {
  try {
    console.log("Testing readSessions() - all, unfiltered...");
    const sessions = await reader.readSessions();
    console.log(`\n✓ Read ${sessions.length} sessions total`);
    
    const ideCount = sessions.filter(s => s.hasIdeActivity).length;
    console.log(`✓ ${ideCount} sessions with IDE activity`);
    console.log(`✓ ${sessions.length - ideCount} CLI-only sessions`);
    
    if (sessions.length > 0) {
      console.log("\nFirst 5 sessions:");
      sessions.slice(0, 5).forEach((s, i) => {
        const ideLabel = s.hasIdeActivity ? ' [IDE+CLI]' : '';
        console.log(`  [${i+1}] ${s.id.slice(0, 8)} | ${s.primaryModel || 'N/A'} | ${(s.totalNanoAIU / 1e9).toFixed(2)} cr${ideLabel}`);
      });
    }
  } catch (err) {
    console.error("❌ Error:", err);
  }
})();
