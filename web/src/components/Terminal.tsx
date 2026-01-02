import { useEffect, useRef, useImperativeHandle, forwardRef } from 'react';
import { Terminal as XTerm } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';

// --- Constants ---

const TERMINAL_THEME = {
    background: '#0d1117', // GitHub Dark Dim
    foreground: '#c9d1d9',
    selectionBackground: '#264f78',
    cursor: '#0d1117', // Hide default cursor (read-only)
};

const FONT_FAMILY = "'JetBrains Mono', 'Consolas', monospace";

/**
 * Handle exposing imperative methods to the parent component.
 */
export interface TerminalHandle {
    /** Write raw string data to the terminal. */
    write: (data: string) => void;
    /** Write a line to the terminal. */
    writeln: (data: string) => void;
    /** Clear the terminal buffer. */
    clear: () => void;
}

interface TerminalProps {
    /** Current execution status ("running" | "idle" | etc) */
    status: string;
}

/**
 * Terminal Component
 * 
 * A robust wrapper around xterm.js providing a read-only output view.
 * Features:
 * - responsive resizing using ResizeObserver
 * - custom theme matching the IDE
 * - copy/paste support for read-only selection
 */
const Terminal = forwardRef<TerminalHandle, TerminalProps>(({ status }, ref) => {
    // Refs
    const mountRef = useRef<HTMLDivElement>(null);
    const xtermRef = useRef<XTerm | null>(null);
    const fitAddonRef = useRef<FitAddon | null>(null);

    // Expose API
    useImperativeHandle(ref, () => ({
        write: (data: string) => xtermRef.current?.write(data),
        writeln: (data: string) => xtermRef.current?.writeln(data),
        clear: () => xtermRef.current?.reset(),
    }));

    useEffect(() => {
        if (!mountRef.current || xtermRef.current) return;

        // 1. Initialize XTerm Instance
        const term = new XTerm({
            cursorBlink: false,
            cursorStyle: 'bar',
            disableStdin: true, // Read-only input
            convertEol: true,
            fontSize: 13,
            lineHeight: 1.5,
            fontFamily: FONT_FAMILY,
            theme: TERMINAL_THEME,
            allowProposedApi: true,
            scrollback: 2000,
        });

        // 2. Attach Fit Addon
        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);
        fitAddonRef.current = fitAddon;

        // 3. Mount to DOM
        term.open(mountRef.current);
        xtermRef.current = term;

        // 4. Custom Key Handler (Enable Copy)
        term.attachCustomKeyEventHandler((e) => {
            // Allow Ctrl+C / Cmd+C if text is selected
            if ((e.ctrlKey || e.metaKey) && e.code === 'KeyC' && term.hasSelection()) {
                return false; // Allow browser default copy behavior
            }
            return true;
        });

        // 5. Robust Resize Logic
        const handleResize = () => {
            try {
                fitAddon.fit();
            } catch (e) {
                // Suppress 'dimensions' error if container is hidden/collapsing
            }
        };

        // Initial fit and observer setup
        requestAnimationFrame(handleResize);
        const resizeObserver = new ResizeObserver(() => requestAnimationFrame(handleResize));
        resizeObserver.observe(mountRef.current);

        // Cleanup
        return () => {
            resizeObserver.disconnect();
            term.dispose();
            xtermRef.current = null;
            fitAddonRef.current = null;
        };
    }, []);

    // Derived Status Styles
    const isRunning = status === 'running';
    const statusColor = isRunning ? '#dbab09' : '#2ea043';
    const statusText = isRunning ? 'Running' : 'Ready';

    return (
        <div style={CONTAINER_STYLE}>
             {/* Hide default scrollbar */}
             <style>{SCROLLBAR_CSS}</style>
            
            {/* Status Header */}
            <div style={STATUS_BAR_STYLE}>
                <span style={STATUS_TITLE_STYLE}>Terminal</span>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <span style={{ 
                        width: '8px', 
                        height: '8px', 
                        borderRadius: '50%', 
                        backgroundColor: statusColor,
                        boxShadow: isRunning ? `0 0 8px ${statusColor}` : 'none',
                        transition: 'all 0.3s'
                    }} />
                    <span style={{ fontSize: '12px', color: statusColor, fontWeight: 500 }}>
                        {statusText}
                    </span>
                </div>
            </div>

            {/* Terminal Mount Point */}
            <div ref={mountRef} style={MOUNT_STYLE} />
        </div>
    );
});

// --- Styles (Extracted for cleanliness) ---

const SCROLLBAR_CSS = `
    .xterm-viewport::-webkit-scrollbar { display: none; }
`;

const CONTAINER_STYLE: React.CSSProperties = {
    width: '100%',
    height: '100%',
    display: 'flex',
    flexDirection: 'column',
    overflow: 'hidden'
};

const STATUS_BAR_STYLE: React.CSSProperties = {
    height: '32px',
    backgroundColor: '#0d1117',
    borderBottom: '1px solid #30363d',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '0 16px',
    flexShrink: 0
};

const STATUS_TITLE_STYLE: React.CSSProperties = {
    fontSize: '12px',
    fontWeight: 600,
    color: '#8b949e',
    textTransform: 'uppercase',
    letterSpacing: '1px'
};

const MOUNT_STYLE: React.CSSProperties = {
    flex: 1,
    width: '100%',
    overflow: 'hidden',
    // Exact padding requested by user
    padding: '4px 0 0 8px', 
};

export default Terminal;