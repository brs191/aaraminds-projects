// Quick test of the session reader compiled code
const reader = require('./out/session/reader');

(async () => {
  try {
    console.log("Testing readThisMonth()...");
    const sessions = await reader.readThisMonth();
    console.log(`✓ Read ${sessions.length} sessions this month`);
    
    if (sessions.length > 0) {
      console.log("\nFirst 3 sessions:");
      sessions.slice(0, 3).forEach((s, i) => {
        console.log(`  [${i+1}] ${s.id} | model=${s.primaryModel} | credits=${(s.totalNanoAIU / 1e9).toFixed(2)} | source=${s.source}`);
      });
    } else {
      console.log("⚠️  No sessions found!");
    }
  } catch (err) {
    console.error("❌ Error:", err);
  }
})();
