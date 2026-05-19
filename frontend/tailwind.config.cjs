module.exports = {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      /* ── 颜色映射至 CSS Token ─────────────────────────────────── */
      colors: {
        brand: {
          DEFAULT: 'var(--color-accent-primary)',
          cyan:    'var(--color-accent-cyan)',
          violet:  'var(--color-accent-violet)',
        },
        surface: {
          page:    'var(--color-bg-page)',
          card:    'var(--color-bg-card)',
          sidebar: 'var(--color-bg-sidebar)',
          input:   'var(--color-bg-input)',
          hover:   'var(--color-bg-hover)',
          active:  'var(--color-bg-active)',
          muted:   'var(--color-bg-muted)',
        },
        content: {
          primary:   'var(--color-text-primary)',
          secondary: 'var(--color-text-secondary)',
          muted:     'var(--color-text-muted)',
          inverse:   'var(--color-text-inverse)',
        },
        line: {
          DEFAULT: 'var(--color-border-default)',
          subtle:  'var(--color-border-subtle)',
          strong:  'var(--color-border-strong)',
        },
        state: {
          success:     'var(--color-success)',
          'success-bg':'var(--color-success-bg)',
          warning:     'var(--color-warning)',
          'warning-bg':'var(--color-warning-bg)',
          danger:      'var(--color-danger)',
          'danger-bg': 'var(--color-danger-bg)',
          info:        'var(--color-info)',
          'info-bg':   'var(--color-info-bg)',
        },
      },
      /* ── 圆角 ─────────────────────────────────────────────────── */
      borderRadius: {
        xs:   'var(--radius-xs)',
        sm:   'var(--radius-sm)',
        md:   'var(--radius-md)',
        lg:   'var(--radius-lg)',
        xl:   'var(--radius-xl)',
        '2xl':'var(--radius-2xl)',
      },
      /* ── 阴影 ─────────────────────────────────────────────────── */
      boxShadow: {
        xs:   'var(--shadow-xs)',
        sm:   'var(--shadow-sm)',
        md:   'var(--shadow-md)',
        lg:   'var(--shadow-lg)',
        xl:   'var(--shadow-xl)',
        '2xl':'var(--shadow-2xl)',
        card: 'var(--shadow-card)',
        'card-hover': 'var(--shadow-card-hover)',
      },
      /* ── 过渡时长 ─────────────────────────────────────────────── */
      transitionDuration: {
        fast:   'var(--duration-fast)',
        normal: 'var(--duration-normal)',
        slow:   'var(--duration-slow)',
      },
    },
  },
  corePlugins: {
    preflight: false
  }
}
