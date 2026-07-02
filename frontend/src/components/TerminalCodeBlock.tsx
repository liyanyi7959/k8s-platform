import React from 'react'

type TerminalCodeBlockProps = {
  title?: React.ReactNode
  content?: string
  className?: string
}

export default function TerminalCodeBlock({ title, content, className }: TerminalCodeBlockProps) {
  return (
    <div className={['app-terminal-code-block', className].filter(Boolean).join(' ')}>
      <div className="app-terminal-code-block__header">
        <div className="app-terminal-code-block__dots" aria-hidden="true">
          <span />
          <span />
          <span />
        </div>
        <span className="app-terminal-code-block__title">{title || 'command.sh'}</span>
      </div>
      <pre className="app-terminal-code-block__body">{content?.trim() || '# 暂无命令'}</pre>
    </div>
  )
}