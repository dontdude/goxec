import { useState } from 'react';
import CodeEditor from './components/CodeEditor';

const INITIAL_CODE = "print('Hello from Goxec!')";

function App() {
  const [code, setCode] = useState<string>(INITIAL_CODE);
  
  const runCode = async () => {
    try {
      console.log("Submitting job...");
      const response = await fetch('http://localhost:8080/submit', {
        method: 'POST',
        headers: {'Content-type': 'application/json'},
        body: JSON.stringify({
          code: code,
          language: 'python',
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to submit job');
      }

      const data = await response.json();
      console.log('Job Submitted ID:', data.job_id);
      alert(`Job Submitted: ${data.job_id}`);
    } catch(error) {
      console.error(error);
      alert('Error submitting job');
    }
  };

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
      <div style={{ display: 'flex', flex: 1 }}>
        {/* Left: Editor */}
        <div style={{ flex: 1, borderRight: '1px solid #333' }}>
          <CodeEditor code={code} onChange={(val) => setCode(val || '')} />
        </div>
        {/* Right: Terminal Placeholder */}
        <div style={{ flex: 1, padding: '10px', fontFamily: 'monospace' }}>
          <p>Terminal Output will appear here...</p>
        </div>
      </div>
    </div>
  );
}

export default App
