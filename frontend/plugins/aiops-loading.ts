import fs from 'node:fs'
import path from 'node:path'

export default (api: any) => {
  const logoData = `data:image/svg+xml;base64,${fs.readFileSync(path.join(api.cwd, 'public/brand/aiops-mark.svg')).toString('base64')}`
  api.register({
    key: 'modifyDevToolLoadingHTML',
    fn: (document: any) => {
    document('html').attr('lang', 'zh-CN')
    document('title').text('AIOPS 智能运维平台 · 正在启动')
    document('#loading').prepend(`
      <img class="aiops-brand-mark" src="${logoData}" alt="AIOPS 智能运维平台" />
      <div class="aiops-brand-name">AIOPS 智能运维</div>
    `)
    document('#loading h3').text('平台服务正在初始化')
    document('#loading small').text('正在加载前端资源，请稍候')
    document('head').append(`
      <style data-aiops-loading>
        :root { color-scheme: light; }
        * { box-sizing: border-box; }
        body {
          margin: 0;
          min-height: 100vh;
          background:
            radial-gradient(circle at 50% 38%, rgba(37, 99, 235, .10), transparent 34%),
            linear-gradient(180deg, #f8fbff 0%, #eef5ff 100%);
          font-family: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;
        }
        #loading {
          width: min(420px, calc(100vw - 40px));
          padding: 38px 42px 32px;
          border: 1px solid rgba(37, 99, 235, .12);
          border-radius: 20px;
          background: rgba(255, 255, 255, .92);
          box-shadow: 0 24px 60px rgba(30, 64, 175, .12);
          text-align: center;
        }
        #loading::before, #loading::after { display: none !important; }
        .aiops-brand-mark {
          display: block;
          width: 64px;
          height: 64px;
          margin: 0 auto 14px;
          filter: drop-shadow(0 12px 18px rgba(37, 99, 235, .22));
        }
        .aiops-brand-name { color: #172554; font-size: 20px; font-weight: 700; letter-spacing: .04em; }
        #loading h3 { margin: 24px 0 8px; color: #1e293b; font-size: 16px; font-weight: 600; }
        #loading small { color: #94a3b8; font-size: 13px; }
        #loading small::after { content: none !important; }
        #loading p.summary {
          margin: 22px 0 8px;
          height: auto;
          color: #2563eb;
          font-family: Inter, sans-serif;
          font-size: 32px;
          font-weight: 700;
        }
        #loading p.summary span { color: #64748b; font-size: 14px; font-weight: 500; }
        #loading p.detail { display: none !important; }
        #loading::after {
          content: "";
          display: block !important;
          width: 100%;
          height: 4px;
          margin-top: 20px;
          border-radius: 999px;
          background: linear-gradient(90deg, #2563eb, #06b6d4, #2563eb);
          background-size: 200% 100%;
          animation: aiops-loading-flow 1.6s linear infinite;
        }
        code.progress-details { display: none; }
        @keyframes aiops-loading-flow { to { background-position: -200% 0; } }
      </style>
    `)
      return document
    },
  })
}
