import { useState, useRef } from 'react';
import CodeEditor from './components/CodeEditor';
import Terminal, { type TerminalHandle } from './components/Terminal';
import { useJobRunner } from './hooks/useJobRunner';

function App() {
  const [language, setLanguage] = useState<string>('javascript');
  const [code, setCode] = useState<string>(LANGUAGE_SNIPPETS['javascript']);
  const termRef = useRef<TerminalHandle>(null);
  
  const { run, status } = useJobRunner({ terminalRef: termRef });

  const handleLanguageChange = (lang: string) => {
      setLanguage(lang);
      setCode(LANGUAGE_SNIPPETS[lang]);
      termRef.current?.clear();
  };

  const handleRun = () => {
      run(code, language);
  };

  const isRunning = status === 'running';

  return (
    <div className="layout-root">
        
        {/* --- Header --- */}
        <header className="header">
            {/* Brand Logo */}
            <div className="brand">
                <span className="brand-icon">&gt;_</span>
                <span className="brand-text">goxec</span>
            </div>

            {/* Controls */}
            <div className="controls-group">
                {/* Language Toggle */}
                <div className="lang-toggle">
                    <button 
                        className={`toggle-item ${language === 'javascript' ? 'active' : ''}`}
                        onClick={() => handleLanguageChange('javascript')}
                        aria-label="Select JavaScript"
                    >
                        JavaScript
                    </button>
                    <button 
                        className={`toggle-item ${language === 'python' ? 'active' : ''}`}
                        onClick={() => handleLanguageChange('python')}
                        aria-label="Select Python"
                    >
                        Python
                    </button>
                </div>

                <div className="header-divider" />

                {/* Run Button */}
                <button 
                    className={`btn-run ${isRunning ? 'disabled' : ''}`}
                    onClick={handleRun}
                    disabled={status === 'queued' || isRunning}
                >
                    {isRunning ? (
                        <>
                            <span className="spinner">⟳</span> Running...
                        </>
                    ) : (
                        <>
                            <span className="play-icon">▶</span> Run Code
                        </>
                    )}
                </button>
            </div>
        </header>

        {/* --- Workspace --- */}
        <main className="main-content">
            <div className="pane-editor">
                <CodeEditor code={code} onChange={(val) => setCode(val || '')} />
            </div>

            <div className="pane-terminal">
                <Terminal ref={termRef} status={status} />
            </div>
        </main>
    </div>
  );
}

// --- Constants & Data ---

const LANGUAGE_SNIPPETS: Record<string, string> = {
  python: `import sys
import time
import platform

print("Goxec Python Runner")

def heavy_computation():
    print("Loading modules...")
    for i in range(3):
        time.sleep(0.3)
        print(".", end="", flush=True)
    print(" Done!")

    print("\\nSystem Stats:")
    print(f"  • Platform: {platform.system()} {platform.release()}")
    print(f"  • Version: {platform.python_version()}")
    print(f"  • Architecture: {platform.machine()}")

if __name__ == "__main__":
    heavy_computation()`,

  javascript: `console.log("Goxec Node.js Runner");

async function heavyComputation() {
    process.stdout.write("Loading modules");
    for(let i=0; i<3; i++) {
        await new Promise(r => setTimeout(r, 300));
        process.stdout.write(".");
    }
    console.log(" Done!");
    
    const mem = (process.memoryUsage().heapUsed / 1024 / 1024).toFixed(2);
    
    console.log("\\nSystem Stats:");
    console.log(\`  • Platform: \${process.platform}\`);
    console.log(\`  • Architecture: \${process.arch}\`);
    console.log(\`  • Memory Usage: \${mem} MB\`);
}

heavyComputation();`
};

export default App;