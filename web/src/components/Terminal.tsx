import { useEffect, useRef, useImperativeHandle, forwardRef } from 'react';
import { Terminal as XTerm } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';

/**
 * Interface for the imperative handle exposed by the Terminal component.
 * Allows parent components to control the terminal instance.
 */
export interface TerminalHandle {
    /** Write data to the terminal. */
    write: (data: string) => void;
    /** Write a line to the terminal. */
    writeln: (data: string) => void;
    /** Clear the terminal. */
    clear: () => void;
}

/**
 * Terminal Component
 * 
 * A wrapper around xterm.js that provides a read-only, dark-themed log viewer.
 * It handles:
 * - Responsive resizing (via FitAddon)
 * - Copy/Paste support (Ctrl+C)
 * - Custom scrollbar styling
 * - Layout shift prevention using a nested container strategy
 */
const Terminal = forwardRef<TerminalHandle>((_, ref) => {
    // 1. Refs for DOM and XTerm instance
    // mountRef: The inner div where XTerm injects the <canvas>
    const mountRef = useRef<HTMLDivElement>(null);
    const xtermRef = useRef<XTerm | null>(null);
    const fitAddonRef = useRef<FitAddon | null>(null);

    // 2. Expose methods to parent
    useImperativeHandle(ref, () => ({
        write: (data: string) => xtermRef.current?.write(data),
        writeln: (data: string) => xtermRef.current?.writeln(data),
        clear: () => xtermRef.current?.reset(),
    }));

    useEffect(() => {
        // Guard: Ensure text selection is possible and preventing re-initialization
        if (!mountRef.current || xtermRef.current) return;

        // 3. Initialize xterm.js
        const term = new XTerm({
            cursorBlink: false,
            cursorStyle: 'bar',
            disableStdin: true, // Read-only
            convertEol: true,
            fontSize: 14,
            fontFamily: 'Consolas, "Courier New", monospace',
            theme: {
                background: '#1e1e1e', // VS Code Dark
                foreground: '#cccccc',
                selectionBackground: '#264f78',
                cursor: '#1e1e1e', // Hidden cursor
            },
            allowProposedApi: true,
        });

        // 4. Attach FitAddon
        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);
        fitAddonRef.current = fitAddon;

        // 5. Mount to DOM
        term.open(mountRef.current);
        xtermRef.current = term;

        // 6. Custom Key Handler: Enable Ctrl+C Copy
        term.attachCustomKeyEventHandler((e) => {
            // Allow browser default copy if Ctrl+C or Cmd+C is pressed
            if ((e.ctrlKey || e.metaKey) && e.code === 'KeyC' && term.hasSelection()) {
                return false;
            }
            return true;
        });

        // 7. Robust Fit Function
        // Suppresses 'dimensions' error caused by race conditions during resize/mount.
        const safeFit = () => {
            if (!fitAddonRef.current || !mountRef.current) return;
            
            // Skip fitting if element is hidden/collapsed
            if (mountRef.current.clientWidth === 0) return;

            try {
                fitAddonRef.current.fit();
            } catch (e) {
                // Benign error: xterm renderer not ready
            }
        };

        // Initial fit (delayed to ensure paint)
        requestAnimationFrame(safeFit);

        // 8. Resize Observer
        const resizeObserver = new ResizeObserver(() => {
            // Debounce fit calls to animation frame
            requestAnimationFrame(safeFit);
        });
        resizeObserver.observe(mountRef.current);

        // Cleanup
        return () => {
            resizeObserver.disconnect();
            term.dispose();
            xtermRef.current = null;
        };
    }, []);

    return (
        // Outer Container: Manages padding and background (Layout)
        <div style={OUTER_STYLE}>
            {/* Scrollbar Styles */}
            <style>{SCROLLBAR_CSS}</style>
            
            {/* Inner Container: Manages XTerm instance (Content) */}
            <div ref={mountRef} style={INNER_STYLE} />
        </div>
    );
});

// --- Styles ---

const OUTER_STYLE: React.CSSProperties = {
    width: '100%',
    height: '100%',
    backgroundColor: '#1e1e1e',
    padding: '12px',
    boxSizing: 'border-box',
    overflow: 'hidden', // Prevent outer scrollbars
    position: 'relative',
};

const INNER_STYLE: React.CSSProperties = {
    width: '100%',
    height: '100%',
    overflow: 'hidden', // Contain XTerm scrollbars
};

const SCROLLBAR_CSS = `
    .xterm-viewport::-webkit-scrollbar { width: 10px; }
    .xterm-viewport::-webkit-scrollbar-track { background: #1e1e1e; }
    .xterm-viewport::-webkit-scrollbar-thumb { background: #424242; border-radius: 0px; }
    .xterm-viewport::-webkit-scrollbar-thumb:hover { background: #4f4f4f; }
`;

export default Terminal;