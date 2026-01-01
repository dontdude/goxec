import { useEffect, useRef, useImperativeHandle, forwardRef } from 'react';
import { Terminal as XTerm } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';

/**
 * TerminalHandle defines the imperative methods exposed to the parent component.
 */
export interface TerminalHandle {
    write: (data: string) => void;
    writeln: (data: string) => void;
    clear: () => void;
}

/**
 * Terminal wraps the xterm.js library into a React component.
 * It provides a ref-based API to interact with the terminal instance.
 */
const Terminal = forwardRef<TerminalHandle>((_, ref) => {
    const terminalRef = useRef<HTMLDivElement>(null);
    const xtermRef = useRef<XTerm | null>(null);

    useImperativeHandle(ref, () => ({
        write: (data: string) => xtermRef.current?.write(data),
        writeln: (data: string) => xtermRef.current?.writeln(data),
        clear: () => xtermRef.current?.reset(),
    }));

    useEffect(() => {
        let fitAddon: FitAddon | null = null;
        let resizeListener: (() => void) | null = null;

        const initTerminal = () => {
             if (xtermRef.current || !terminalRef.current) return;
             if (terminalRef.current.clientWidth === 0 || terminalRef.current.clientHeight === 0) return;

             const term = new XTerm({
                cursorBlink: true,
                theme: {
                    background: '#1e1e1e',
                    foreground: '#cccccc',
                },
                fontSize: 14,
                fontFamily: 'Consolas, "Courier New", monospace',
                allowProposedApi: true,
            });

            fitAddon = new FitAddon();
            term.loadAddon(fitAddon);

            term.open(terminalRef.current);
            try {
                fitAddon.fit();
            } catch (e) {
                console.warn("Initial fit failed", e);
            }
            
            xtermRef.current = term;

             // Handle window resizes
             resizeListener = () => {
                 try {
                     fitAddon?.fit();
                 } catch (e) {
                     console.warn("Resize fit failed", e);
                 }
             };
             window.addEventListener('resize', resizeListener);
        };

        const resizeObserver = new ResizeObserver(() => {
            if (!xtermRef.current) {
                initTerminal();
            } else {
                try {
                    fitAddon?.fit();
                } catch(e) {}
            }
        });
        
        if (terminalRef.current) {
            resizeObserver.observe(terminalRef.current);
        }

        return () => {
            resizeObserver.disconnect();
            if (resizeListener) {
                window.removeEventListener('resize', resizeListener);
            }
            if (xtermRef.current) {
                xtermRef.current.dispose();
                xtermRef.current = null;
            }
        };
    }, []);

    return (
        <div 
            ref={terminalRef} 
            style={{ width: '100%', height: '100%', paddingLeft: '10px' }} 
        />
    );
});

export default Terminal;