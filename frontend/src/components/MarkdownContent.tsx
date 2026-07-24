import React from 'react'
import { message } from 'antd'
import { CopyOutlined } from '@ant-design/icons'

type MarkdownTone = 'light' | 'dark'

type MarkdownBlock =
  | { type: 'paragraph'; lines: string[] }
  | { type: 'heading'; level: number; content: string }
  | { type: 'unordered-list'; items: string[] }
  | { type: 'ordered-list'; items: string[] }
  | { type: 'blockquote'; lines: string[] }
  | { type: 'code'; language: string; content: string }
  | { type: 'table'; headers: string[]; rows: string[][] }
  | { type: 'rule' }

type MarkdownContentProps = {
  content?: string
  tone?: MarkdownTone
}

const headingSizeMap: Record<number, number> = {
  1: 28,
  2: 24,
  3: 20,
  4: 18,
  5: 16,
  6: 15,
}

function parseMarkdownBlocks(content: string): MarkdownBlock[] {
  const normalized = content.replace(/\r\n/g, '\n').trim()
  if (!normalized) {
    return []
  }

  const lines = normalized.split('\n')
  const blocks: MarkdownBlock[] = []
  let paragraphLines: string[] = []

  const flushParagraph = () => {
    if (paragraphLines.length === 0) {
      return
    }
    blocks.push({
      type: 'paragraph',
      lines: paragraphLines,
    })
    paragraphLines = []
  }

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index] || ''
    const trimmed = line.trim()

    if (!trimmed) {
      flushParagraph()
      continue
    }

    const codeFenceMatch = trimmed.match(/^```([\w-]+)?\s*$/)
    if (codeFenceMatch) {
      flushParagraph()
      const codeLines: string[] = []
      const language = codeFenceMatch[1] || ''
      index += 1
      while (index < lines.length) {
        const currentLine = lines[index] || ''
        if (currentLine.trim().match(/^```$/)) {
          break
        }
        codeLines.push(currentLine)
        index += 1
      }
      blocks.push({
        type: 'code',
        language,
        content: codeLines.join('\n'),
      })
      continue
    }

    const headingMatch = trimmed.match(/^(#{1,6})\s+(.+)$/)
    if (headingMatch) {
      flushParagraph()
      blocks.push({
        type: 'heading',
        level: headingMatch[1]?.length || 1,
        content: headingMatch[2] || '',
      })
      continue
    }

    if (/^(-{3,}|\*{3,}|_{3,})$/.test(trimmed)) {
      flushParagraph()
      blocks.push({ type: 'rule' })
      continue
    }

    const unorderedListMatch = trimmed.match(/^[-*+]\s+(.+)$/)
    if (unorderedListMatch) {
      flushParagraph()
      const items: string[] = [unorderedListMatch[1] || '']
      while (index + 1 < lines.length) {
        const nextLine = (lines[index + 1] || '').trim()
        const nextMatch = nextLine.match(/^[-*+]\s+(.+)$/)
        if (!nextMatch) {
          break
        }
        items.push(nextMatch[1] || '')
        index += 1
      }
      blocks.push({
        type: 'unordered-list',
        items,
      })
      continue
    }

    const orderedListMatch = trimmed.match(/^\d+\.\s+(.+)$/)
    if (orderedListMatch) {
      flushParagraph()
      const items: string[] = [orderedListMatch[1] || '']
      while (index + 1 < lines.length) {
        const nextLine = (lines[index + 1] || '').trim()
        const nextMatch = nextLine.match(/^\d+\.\s+(.+)$/)
        if (!nextMatch) {
          break
        }
        items.push(nextMatch[1] || '')
        index += 1
      }
      blocks.push({
        type: 'ordered-list',
        items,
      })
      continue
    }

    const blockquoteMatch = trimmed.match(/^>\s?(.*)$/)
    if (blockquoteMatch) {
      flushParagraph()
      const quoteLines: string[] = [blockquoteMatch[1] || '']
      while (index + 1 < lines.length) {
        const nextLine = (lines[index + 1] || '').trim()
        const nextMatch = nextLine.match(/^>\s?(.*)$/)
        if (!nextMatch) {
          break
        }
        quoteLines.push(nextMatch[1] || '')
        index += 1
      }
      blocks.push({
        type: 'blockquote',
        lines: quoteLines,
      })
      continue
    }

    // Markdown 表格：| col1 | col2 | 后跟 |---|---|
    if (trimmed.startsWith('|') && trimmed.endsWith('|') && index + 1 < lines.length) {
      const separatorLine = (lines[index + 1] || '').trim()
      if (/^\|[\s:|-]+\|$/.test(separatorLine) && separatorLine.includes('-')) {
        flushParagraph()
        const parseRow = (line: string): string[] => {
          const cells = line.split('|').map((c) => c.trim())
          // 去掉首尾空元素（因为行首行尾有 |）
          if (cells.length > 0 && cells[0] === '') cells.shift()
          if (cells.length > 0 && cells[cells.length - 1] === '') cells.pop()
          return cells
        }
        const headers = parseRow(trimmed)
        const rows: string[][] = []
        index += 2 // 跳过分隔行
        while (index < lines.length) {
          const rowLine = (lines[index] || '').trim()
          if (!rowLine.startsWith('|') || !rowLine.endsWith('|')) {
            break
          }
          rows.push(parseRow(rowLine))
          index += 1
        }
        index -= 1 // for 循环会 +1
        blocks.push({ type: 'table', headers, rows })
        continue
      }
    }

    paragraphLines.push(line)
  }

  flushParagraph()
  return blocks
}

function renderPlainText(text: string, keyPrefix: string): React.ReactNode[] {
  const lines = text.split('\n')
  const nodes: React.ReactNode[] = []
  lines.forEach((line, index) => {
    if (line) {
      nodes.push(
        <React.Fragment key={`${keyPrefix}-text-${index}`}>
          {line}
        </React.Fragment>,
      )
    }
    if (index < lines.length - 1) {
      nodes.push(<br key={`${keyPrefix}-br-${index}`} />)
    }
  })
  return nodes
}

function renderInline(text: string, tone: MarkdownTone, keyPrefix: string): React.ReactNode[] {
  const nodes: React.ReactNode[] = []
  const pattern = /(`[^`\n]+`|\*\*[^*\n]+?\*\*|\[[^\]]+\]\((https?:\/\/[^)\s]+)\))/g
  let lastIndex = 0

  for (const match of text.matchAll(pattern)) {
    const token = match[0] || ''
    const start = match.index || 0
    if (start > lastIndex) {
      nodes.push(...renderPlainText(text.slice(lastIndex, start), `${keyPrefix}-${start}`))
    }

    if (token.startsWith('`') && token.endsWith('`')) {
      nodes.push(
        <code
          key={`${keyPrefix}-code-${start}`}
          style={{
            fontFamily: 'Consolas, Monaco, monospace',
            background: tone === 'dark' ? 'rgba(255,255,255,0.1)' : '#f0f0f0',
            borderRadius: 6,
            padding: '2px 6px',
            color: tone === 'dark' ? '#ff9c6e' : '#c41d7f',
          }}
        >
          {token.slice(1, -1)}
        </code>,
      )
    } else if (token.startsWith('**') && token.endsWith('**')) {
      nodes.push(
        <strong key={`${keyPrefix}-strong-${start}`}>
          {token.slice(2, -2)}
        </strong>,
      )
    } else {
      const linkMatch = token.match(/^\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)$/)
      if (linkMatch) {
        nodes.push(
          <a
            key={`${keyPrefix}-link-${start}`}
            href={linkMatch[2]}
            target="_blank"
            rel="noopener noreferrer"
            style={{
              color: tone === 'dark' ? '#dbeafe' : '#2563eb',
              textDecoration: 'underline',
            }}
          >
            {linkMatch[1]}
          </a>,
        )
      } else {
        nodes.push(...renderPlainText(token, `${keyPrefix}-fallback-${start}`))
      }
    }

    lastIndex = start + token.length
  }

  if (lastIndex < text.length) {
    nodes.push(...renderPlainText(text.slice(lastIndex), `${keyPrefix}-tail`))
  }

  return nodes
}

function highlightCode(code: string, language: string): React.ReactNode {
  // 关键字高亮，按语言区分
  const keywords = language === 'yaml'
    ? /\b(apiVersion|kind|metadata|name|namespace|spec|replicas|selector|template|containers|image|ports|env|volumeMounts|volumes|resources|limits|requests)\b/g
    : language === 'json'
    ? /\b(true|false|null)\b/g
    : language === 'go'
    ? /\b(func|return|if|else|for|range|type|struct|interface|package|import|var|const|nil|map|chan|go|defer)\b/g
    : /\b(if|then|fi|for|do|done|case|esac|while|echo|export|local|return|function)\b/g

  const lines = code.split('\n')
  return lines.map((line, lineIdx) => {
    // 注释行（# 开头）
    const commentMatch = line.match(/^(\s*)(#.*)$/)
    if (commentMatch) {
      return <div key={lineIdx}>{commentMatch[1]}<span style={{ color: '#64748b' }}>{commentMatch[2]}</span></div>
    }
    // 逐字符解析：匹配字符串、关键字、数字
    const parts: React.ReactNode[] = []
    let remaining = line
    let key = 0
    while (remaining.length > 0) {
      const strMatch = remaining.match(/^["'`]([^"'`]+)["'`]/)
      if (strMatch) {
        parts.push(<span key={key++} style={{ color: '#86efac' }}>{strMatch[0]}</span>)
        remaining = remaining.slice(strMatch[0].length)
        continue
      }
      const kwMatch = remaining.match(keywords)
      if (kwMatch && kwMatch.index !== undefined) {
        if (kwMatch.index > 0) {
          parts.push(<span key={key++}>{remaining.slice(0, kwMatch.index)}</span>)
        }
        parts.push(<span key={key++} style={{ color: '#c084fc', fontWeight: 600 }}>{kwMatch[0]}</span>)
        remaining = remaining.slice(kwMatch.index + kwMatch[0].length)
        continue
      }
      // 数字
      const numMatch = remaining.match(/^\d+/)
      if (numMatch) {
        parts.push(<span key={key++} style={{ color: '#fbbf24' }}>{numMatch[0]}</span>)
        remaining = remaining.slice(numMatch[0].length)
        continue
      }
      parts.push(<span key={key++}>{remaining[0]}</span>)
      remaining = remaining.slice(1)
    }
    return <div key={lineIdx}>{parts}{lineIdx < lines.length - 1 ? '\n' : ''}</div>
  })
}

function renderBlock(block: MarkdownBlock, tone: MarkdownTone, index: number) {
  const textColor = tone === 'dark' ? 'inherit' : '#141414'
  const secondaryBorder = tone === 'dark' ? 'rgba(255,255,255,0.18)' : '#e5e7eb'
  const blockquoteBackground = tone === 'dark' ? 'rgba(255,255,255,0.08)' : '#f8fafc'

  switch (block.type) {
    case 'heading':
      return (
        <div
          key={`heading-${index}`}
          style={{
            fontSize: headingSizeMap[block.level] || 15,
            lineHeight: 1.5,
            fontWeight: 700,
            margin: index === 0 ? 0 : '16px 0 8px',
            color: textColor,
          }}
        >
          {renderInline(block.content, tone, `heading-${index}`)}
        </div>
      )
    case 'unordered-list':
      return (
        <ul
          key={`ul-${index}`}
          style={{
            margin: '8px 0 12px 20px',
            padding: 0,
            color: textColor,
          }}
        >
          {block.items.map((item, itemIndex) => (
            <li key={`ul-${index}-${itemIndex}`} style={{ marginBottom: 6, lineHeight: 1.7 }}>
              {renderInline(item, tone, `ul-${index}-${itemIndex}`)}
            </li>
          ))}
        </ul>
      )
    case 'ordered-list':
      return (
        <ol
          key={`ol-${index}`}
          style={{
            margin: '8px 0 12px 20px',
            padding: 0,
            paddingLeft: 22,
            listStyleType: 'decimal',
            color: textColor,
          }}
        >
          {block.items.map((item, itemIndex) => (
            <li key={`ol-${index}-${itemIndex}`} style={{ marginBottom: 6, lineHeight: 1.7 }}>
              {renderInline(item, tone, `ol-${index}-${itemIndex}`)}
            </li>
          ))}
        </ol>
      )
    case 'blockquote':
      return (
        <blockquote
          key={`quote-${index}`}
          style={{
            margin: '12px 0',
            padding: '10px 14px',
            borderLeft: `4px solid ${secondaryBorder}`,
            background: blockquoteBackground,
            borderRadius: 10,
            color: textColor,
          }}
        >
          {renderInline(block.lines.join('\n'), tone, `quote-${index}`)}
        </blockquote>
      )
    case 'code':
      return (
        <div key={`code-${index}`} style={{ margin: '12px 0', borderRadius: 12, overflow: 'hidden', border: '1px solid #1e293b' }}>
          {/* 语言标签 + 复制按钮 */}
          {block.language && (
            <div style={{
              background: '#1e293b', color: '#94a3b8',
              fontSize: 11, padding: '4px 12px',
              fontFamily: 'monospace',
              display: 'flex', justifyContent: 'space-between', alignItems: 'center',
            }}>
              <span>{block.language}</span>
              <CopyOutlined onClick={() => { navigator.clipboard.writeText(block.content); message.success('已复制') }} style={{ cursor: 'pointer' }} />
            </div>
          )}
          <pre style={{
            margin: 0, padding: '14px 16px',
            overflowX: 'auto',
            background: '#0f172a',
            color: '#e5e7eb',
            fontSize: 13, lineHeight: 1.65,
            fontFamily: 'Consolas, Monaco, monospace',
          }}>
            {highlightCode(block.content, block.language)}
          </pre>
        </div>
      )
    case 'rule':
      return (
        <div
          key={`rule-${index}`}
          style={{
            margin: '16px 0',
            borderTop: `1px solid ${secondaryBorder}`,
          }}
        />
      )
    case 'table':
      return (
        <div key={`table-${index}`} style={{ margin: '12px 0', overflowX: 'auto' }}>
          <table style={{
            borderCollapse: 'collapse',
            width: '100%',
            fontSize: 13,
            lineHeight: 1.6,
          }}>
            <thead>
              <tr>
                {block.headers.map((header, hIdx) => (
                  <th key={`th-${index}-${hIdx}`} style={{
                    border: `1px solid ${secondaryBorder}`,
                    padding: '8px 12px',
                    textAlign: 'left',
                    fontWeight: 600,
                    background: tone === 'dark' ? 'rgba(255,255,255,0.08)' : '#f8fafc',
                    color: textColor,
                  }}>
                    {renderInline(header, tone, `th-${index}-${hIdx}`)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {block.rows.map((row, rIdx) => (
                <tr key={`tr-${index}-${rIdx}`}>
                  {row.map((cell, cIdx) => (
                    <td key={`td-${index}-${rIdx}-${cIdx}`} style={{
                      border: `1px solid ${secondaryBorder}`,
                      padding: '8px 12px',
                      color: textColor,
                    }}>
                      {renderInline(cell, tone, `td-${index}-${rIdx}-${cIdx}`)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )
    case 'paragraph':
    default:
      return (
        <div
          key={`paragraph-${index}`}
          style={{
            margin: index === 0 ? 0 : '0 0 12px',
            color: textColor,
            lineHeight: 1.8,
          }}
        >
          {renderInline(block.lines.join('\n'), tone, `paragraph-${index}`)}
        </div>
      )
  }
}

export default function MarkdownContent({ content, tone = 'light' }: MarkdownContentProps) {
  const blocks = parseMarkdownBlocks(content || '')
  if (blocks.length === 0) {
    return null
  }

  return (
    <div>
      {blocks.map((block, index) => renderBlock(block, tone, index))}
    </div>
  )
}
