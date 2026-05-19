import { getToken } from '@/shared/utils/auth'

export type TerminalUiTheme = 'dark' | 'light'

function normalizeBaseUrl(input: string): string {
	const raw = String(input ?? '').trim()
	if (!raw) return ''
	try {
		const url = new URL(raw.startsWith('http') ? raw : `http://${raw}`)
		return `${url.protocol}//${url.host}${url.pathname.replace(/\/+$/, '')}`
	} catch {
		return ''
	}
}

export function resolveTerminalBaseUrl(): string {
	const runtime = normalizeBaseUrl(localStorage.getItem('k8s_platform_api_base') ?? '')
	if (runtime) return runtime

	const envBase = normalizeBaseUrl(import.meta.env.VITE_API_BASE_URL ?? '')
	if (envBase) return envBase

	// 默认走当前页面同源地址，让 Vite dev server 的 /api WebSocket 代理接管转发。
	// 这样可以避免前端 HTTP 走代理成功、终端 WS 却被强制直连 8080 导致连接失败。
	return window.location.origin
}

export function buildTerminalWebSocketUrl(pathname: string, params?: Record<string, string | number | undefined | null>): string {
	const url = new URL(resolveTerminalBaseUrl())
	url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
	url.pathname = pathname.startsWith('/') ? pathname : `/${pathname}`

	Object.entries(params ?? {}).forEach(([key, value]) => {
		if (value === undefined || value === null || value === '') return
		url.searchParams.set(key, String(value))
	})

	const token = getToken()
	if (token && !url.searchParams.get('token')) {
		url.searchParams.set('token', token)
	}
	return url.toString()
}

export function normalizePastedText(text: string): string {
	return String(text ?? '').replace(/\r\n/g, '\n')
}

export function getTerminalTheme(theme: TerminalUiTheme) {
	if (theme === 'light') {
		return {
			background: '#f8fafc',
			foreground: '#0f172a',
			cursor: '#2563eb',
			cursorAccent: '#f8fafc',
			selectionBackground: 'rgba(37, 99, 235, 0.18)',
			selectionInactiveBackground: 'rgba(148, 163, 184, 0.18)',
			black: '#0f172a',
			red: '#dc2626',
			green: '#16a34a',
			yellow: '#ca8a04',
			blue: '#2563eb',
			magenta: '#9333ea',
			cyan: '#0891b2',
			white: '#e2e8f0',
			brightBlack: '#475569',
			brightRed: '#ef4444',
			brightGreen: '#22c55e',
			brightYellow: '#eab308',
			brightBlue: '#60a5fa',
			brightMagenta: '#c084fc',
			brightCyan: '#22d3ee',
			brightWhite: '#ffffff'
		}
	}

	return {
		background: '#0b1220',
		foreground: '#dbe7ff',
		cursor: '#7cc4ff',
		cursorAccent: '#0b1220',
		selectionBackground: 'rgba(124, 196, 255, 0.22)',
		selectionInactiveBackground: 'rgba(71, 85, 105, 0.24)',
		black: '#1e293b',
		red: '#ef4444',
		green: '#22c55e',
		yellow: '#eab308',
		blue: '#60a5fa',
		magenta: '#c084fc',
		cyan: '#22d3ee',
		white: '#e2e8f0',
		brightBlack: '#475569',
		brightRed: '#f87171',
		brightGreen: '#4ade80',
		brightYellow: '#facc15',
		brightBlue: '#93c5fd',
		brightMagenta: '#d8b4fe',
		brightCyan: '#67e8f9',
		brightWhite: '#f8fafc'
	}
}
