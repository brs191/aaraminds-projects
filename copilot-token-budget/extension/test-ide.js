// Test IDE session collection
const reader = require('./out/session/reader');

(async () => {
  try {
    console.log("Testing collectIdeSessions()...");
    const sessions = await reader.collectIdeSessions();
    console.log(`✓ Read ${sessions.length} IDE sessions`);
    
    if (sessions.length > 0) {
      console.log("\nFirst 3 IDE sessions:");
      sessions.slice(0, 3).forEach((s, i) => {
        console.log(`  [${i+1}] ${s.id} | model=${s.primaryModel} | credits=${(s.totalNanoAIU / 1e9).toFixed(2)}`);
      });
    } else {
      console.log("⚠️  No IDE sessions found - checking if IDE user data roots exist...");
      
      // Check the paths
      const os = require('os');
      const path = require('path');
      const fs = require('fs');
      
      const home = os.homedir();
      const ideRoots = [
        path.join(home, 'Library', 'Application Support', 'Code', 'User'),
        path.join(home, 'Library', 'Application Support', 'Code - Insiders', 'User'),
      ];
      
      console.log("\nChecking IDE user data roots:");
      for (const root of ideRoots) {
        const exists = fs.existsSync(root);
        console.log(`  ${exists ? '✓' : '✗'} ${root}`);
      }
    }
  } catch (err) {
    console.error("❌ Error:", err);
  }
})();
