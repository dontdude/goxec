import { useRef } from 'react';
import type { TerminalHandle } from '../components/Terminal';

interface UseJobRunnerProps {
    code: string;
    terminalRef: React.RefObject<TerminalHandle | null>;
}

export const useJobRunner = ({ code, terminalRef }: UseJobRunnerProps) => {
    // Keep track of active socket to close it if needed (optional cleanup)
    const wsRef = useRef<WebSocket | null>(null);

    const runCode = async () => {
        // Reset Terminal state
        termClear();
        termPrintln("Compiling and Scheduling Job...");

        try {
            // 1. Submit Job via HTTP
            const response = await fetch('http://localhost:8080/submit', {
                method: 'POST',
                headers: { 'Content-type': 'application/json' },
                body: JSON.stringify({
                    code: code,
                    language: 'python',
                }),
            });

            if (!response.ok) {
                throw new Error(`Submission failed: ${response.statusText}`);
            }

            const data = await response.json();
            const jobID = data.job_id;

            termPrintln(`Job Queued (ID: ${jobID})`);
            termPrintln("--> Connecting to Live Logs...");

            // 2. Connect to WebSocket for Real-time Logs
            connectWebSocket(jobID);

        } catch (error) {
            console.error(error);
            termPrintln(`\r\n[Fatal Error] ${error}`);
            alert('Error submitting job');
        }
    };

    const connectWebSocket = (jobID: string) => {
        if (wsRef.current) {
            wsRef.current.close();
        }

        const ws = new WebSocket(`ws://localhost:8080/ws?job_id=${jobID}`);
        wsRef.current = ws;

        ws.onopen = () => {
            // Connection established
        };

        ws.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                if (msg.type === 'log') {
                    // Convert backend newlines to CRLF for xterm.js
                    const formattedOutput = msg.output.replace(/\n/g, '\r\n');
                    termPrint(formattedOutput);
                }
            } catch (e) {
                console.error("Invalid log format", e);
            }
        };

        ws.onclose = () => {
            termPrintln("\r\n--- Connection Closed ---");
        };

        ws.onerror = (err) => {
            termPrintln(`\r\n[Error] WebSocket Error`);
            console.error(err);
        };
    };

    // Helper to clear terminal
    const termClear = () => {
        terminalRef.current?.clear();
    };

    // Helper to safely write to terminal (with newline)
    const termPrintln = (msg: string) => {
        terminalRef.current?.writeln(msg);
    };

    // Helper to safely write to terminal (raw, no newline)
    const termPrint = (msg: string) => {
        terminalRef.current?.write(msg);
    };

    return { runCode };
};
