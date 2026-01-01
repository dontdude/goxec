import { useState, useRef } from 'react';
import CodeEditor from './components/CodeEditor';
import Terminal, { type TerminalHandle } from './components/Terminal';
import { useJobRunner } from './hooks/useJobRunner';

const LANGUAGE_SNIPPETS: Record<string, string> = {
  python: `print("Hello from Goxec!")
import time
for i in range(5):
    print(f"Processing step {i+1}...")
    time.sleep(0.5)
print("Done.")`,
  javascript: `console.log("Hello from Node.js!");
const start = Date.now();
setTimeout(() => {
    console.log("Waited 2 seconds...");
    console.log("Done in " + (Date.now() - start) + "ms");
}, 2000);`
};

function App() {
  const [language, setLanguage] = useState<string>('javascript');
  const [code, setCode] = useState<string>(LANGUAGE_SNIPPETS['javascript']);
  const termRef = useRef<TerminalHandle>(null);
  
  const { run, status } = useJobRunner({ terminalRef: termRef });

  const handleRun = () => {
      run(code, language);
  };

  const handleLanguageChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
      const newLang = e.target.value;
      setLanguage(newLang);
      setCode(LANGUAGE_SNIPPETS[newLang]);
  };

  return (
    <>
      <header className="header">
        <div className="logo">GOXEC</div>
        <div style={{ display: 'flex', gap: '10px' }}>
            <select 
                value={language} 
                onChange={handleLanguageChange}
                style={{
                    padding: '5px',
                    borderRadius: '4px',
                    backgroundColor: '#333',
                    color: 'white',
                    border: '1px solid #454545'
                }}
            >
                <option value="python">Python</option>
                <option value="javascript">JavaScript (Node.js)</option>
            </select>
            <button 
                className="btn-run" 
                onClick={handleRun}
                disabled={status === 'queued' || status === 'running'}
            >
            {status === 'running' ? 'Running...' : 'Run Code'}
            </button>
        </div>
      </header>

      <div className="main-layout">
        {/* Editor Pane (Left/Top) */}
        <div className="editor-pane">
          <div className="pane-header">Editor - {language === 'python' ? 'main.py' : 'index.js'}</div>
          <CodeEditor code={code} onChange={(val) => setCode(val || '')} />
        </div>

        {/* Terminal Pane (Right/Bottom) */}
        <div className="terminal-pane">
            <div className="pane-header">Terminal</div>
            <Terminal ref={termRef} />
        </div>
      </div>
    </>
  );
}

export default App;