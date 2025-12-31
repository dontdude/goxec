import React from 'react';
import Editor from '@monaco-editor/react';

interface CodeEditorProps {
    code: string;
    onChange: (value: string | undefined) => void;
}

const CodeEditor: React.FC<CodeEditorProps> = ({ code, onChange}) => {
    return (
        <div style={{ height: '100%', width: '100%' }}>
            <Editor
                height="100%"
                defaultLanguage='python'
                theme="vs-dark"
                value={code}
                onChange={onChange}
                options={{
                    minimap: {enabled: false},
                    fontSize: 14,
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                }}
            />
        </div>
    );
};

export default CodeEditor;