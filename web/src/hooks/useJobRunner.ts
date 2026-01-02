import { useRef, useState, useCallback } from 'react';
import type { TerminalHandle } from '../components/Terminal';

interface UseJobRunnerProps {
    terminalRef: React.RefObject<TerminalHandle | null>;
}

export type JobStatus = 'idle' | 'queued' | 'running' | 'completed' | 'error';

export const useJobRunner = ({ terminalRef }: UseJobRunnerProps) => {
    const [status, setStatus] = useState<JobStatus>('idle');
    const wsRef = useRef<WebSocket | null>(null);

    const termPrint = useCallback((msg: string) => {
        terminalRef.current?.write(msg);
    }, [terminalRef]);

    const termPrintln = useCallback((msg: string) => {
        terminalRef.current?.writeln(msg);
    }, [terminalRef]);

    const connectWebSocket = useCallback((jobID: string) => {
        if (wsRef.current) {
            wsRef.current.close();
        }

        const ws = new WebSocket(`ws://localhost:8080/ws?job_id=${jobID}`);
        wsRef.current = ws;

        ws.onopen = () => {
            setStatus('running');
        };

        ws.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);

                if (msg.type === 'log') {
                    // Log: Print output to terminal
                    // Normalize newlines (\n -> \r\n) for xterm.js
                    const formatted = (msg.output || '').replace(/\n/g, '\r\n');
                    termPrint(formatted);
                }
                else if (msg.type === 'status') {
                    // Status: Handle completion signal
                    const finalStatus = msg.status; // "completed" or "failed"

                    if (finalStatus === 'completed') {
                        termPrintln("\r\n\x1b[32m✔ Job Completed Successfully\x1b[0m");
                    } else if (finalStatus === 'failed') {
                        termPrintln("\r\n\x1b[31m✘ Job Failed\x1b[0m");
                    }

                    // Reset state to 'idle' so user can run again
                    setStatus('idle');

                    // Close the socket as the job is done
                    ws.close();
                }
            } catch (e) {
                console.error("Failed to parse WebSocket message", e);
            }
        };

        ws.onclose = () => {
            // We logged completion above, so just specific "Session Ended"
            // Using grey color for system messages
            termPrintln("\r\n\x1b[90m--- Session Ended ---\x1b[0m");

            // Safety: Ensure we go back to idle if socket closes unexpectedly
            setStatus((prev) => (prev === 'running' ? 'idle' : prev));
        };

        ws.onerror = () => {
            termPrintln(`\r\n\x1b[31m[System] WebSocket Connection Error\x1b[0m`);
            setStatus('error');
            // Allow retry
            setTimeout(() => setStatus('idle'), 2000);
        };
    }, [termPrint, termPrintln]);

    const run = useCallback(async (code: string, language: string) => {
        if (status === 'running' || status === 'queued') return;

        terminalRef.current?.clear();
        setStatus('queued');
        termPrintln(`\x1b[36m--> Scheduling Job (${language})...\x1b[0m`);

        try {
            const response = await fetch('http://localhost:8080/submit', {
                method: 'POST',
                headers: { 'Content-type': 'application/json' },
                body: JSON.stringify({ code, language }),
            });

            if (!response.ok) {
                throw new Error(`API Error: ${response.statusText}`);
            }

            const data = await response.json();
            termPrintln(`\x1b[32m--> Job Queued ID: ${data.job_id}\x1b[0m`);
            termPrintln("\x1b[90m--> Connecting to log stream...\x1b[0m\r\n");

            connectWebSocket(data.job_id);

        } catch (error) {
            console.error(error);
            termPrintln(`\r\n\x1b[31;1m[Fatal Error] Failed to submit job: ${error}\x1b[0m`);
            setStatus('idle');
        }
    }, [status, terminalRef, connectWebSocket, termPrintln]);

    return {
        run,
        status
    };
};
