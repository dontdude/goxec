import { useState, useRef } from 'react';
import CodeEditor from './components/CodeEditor';
import Terminal, { type TerminalHandle } from './components/Terminal';
import { useJobRunner } from './hooks/useJobRunner';

const INITIAL_CODE = "print('Hello from Goxec!')";

function App() {
  const [code, setCode] = useState<string>(INITIAL_CODE);
  const termRef = useRef<TerminalHandle>(null);
  
  const { runCode } = useJobRunner({ code, terminalRef: termRef });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh', backgroundColor: '#1e1e1e', color: 'white' }}>
      {/* Header */}
      <div style={{ padding: '10px', display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid #333' }}>
        <h3>Goxec Remote Runner</h3>
        <button onClick={runCode} style={{ padding: '8px 16px', backgroundColor: '#4caf50', color: 'white', border: 'none', cursor: 'pointer' }}>
          Run Code
        </button>
      </div>

      {/* Main Content: Split Screen */}
      <div style={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        {/* Left: Editor (50%) */}
        <div style={{ flex: 1, borderRight: '1px solid #333' }}>
          <CodeEditor code={code} onChange={(val) => setCode(val || '')} />
        </div>

        {/* Right: XTerm Console (50%) */}
        <div style={{ flex: 1, backgroundColor: '#1e1e1e' }}>
          <Terminal ref={termRef} />
        </div>
      </div>
    </div>
  );
}

export default App;