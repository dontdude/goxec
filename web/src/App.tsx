import { useState, useRef } from 'react';
import CodeEditor from './components/CodeEditor';
import Terminal, { type TerminalHandle } from './components/Terminal';
import { useJobRunner } from './hooks/useJobRunner';

const INITIAL_CODE = `print("Hello from Goxec!")
import time
for i in range(5):
    print(f"Processing step {i+1}...")
    time.sleep(0.5)
print("Done.")`;

function App() {
  const [code, setCode] = useState<string>(INITIAL_CODE);
  const termRef = useRef<TerminalHandle>(null);
  
  const { run, status } = useJobRunner({ terminalRef: termRef });

  const handleRun = () => {
      run(code, 'python');
  };

  return (
    <>
      <header className="header">
        <div className="logo">GOXEC</div>
        <button 
            className="btn-run" 
            onClick={handleRun}
            disabled={status === 'queued' || status === 'running'}
        >
          {status === 'running' ? 'Running...' : 'Run Code'}
        </button>
      </header>

      <div className="main-layout">
        {/* Editor Pane (Left/Top) */}
        <div className="editor-pane">
          <div className="pane-header">Editor - main.py</div>
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