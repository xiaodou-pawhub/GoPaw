import { createI18n } from 'vue-i18n'

const messages = {
  'zh-CN': {
    common: {
      loading: '加载中...',
      save: '保存',
      cancel: '取消',
      delete: '删除',
      edit: '编辑',
      confirm: '确认',
      success: '成功',
      error: '错误',
      action: '操作',
      back: '返回',
      next: '下一步',
      finish: '完成'
    },
    nav: {
      chat: '聊天',
      settings: '设置',
      providers: 'LLM 提供商',
      channels: '频道',
      agent: 'Agent 设定',
      cron: '定时任务',
      logs: '系统日志',
      skills: '技能管理'
    },
    chat: {
      title: '聊天',
      placeholder: '输入消息...',
      send: '发送',
      newChat: '新对话',
      history: '历史会话',
      thinking: '思考中...',
      welcome: '你好！我是 GoPaw，你的 AI 助理。有什么可以帮你的吗？',
      deleteConfirm: '删除后无法恢复该会话的所有消息，是否继续？'
    },
    settings: {
      title: '设置',
      description: '管理系统核心配置与接入能力',
      syncStatus: '已同步至云端',
      modifiedStatus: '未保存的修改',
      markdownTip: '支持 Markdown 语法，设定将作为 System Prompt 注入对话上下文',
      providers: {
        title: 'LLM 提供商',
        description: '配置并切换不同的大模型服务商，支持 OpenAI 格式的 API 接入',
        add: '添加提供商',
        edit: '编辑提供商',
        name: '名称',
        baseURL: 'API 地址',
        apiKey: 'API Key',
        model: '模型',
        active: '活跃',
        setActive: '设为活跃',
        deleteConfirm: '确认删除此提供商吗？',
        noProviders: '暂无 LLM 提供商',
        addFirst: '请添加第一个 LLM 提供商以开始使用',
        placeholder: {
          name: '例如：OpenAI',
          baseURL: 'https://api.openai.com/v1',
          apiKey: 'sk-...',
          model: 'gpt-4o-mini'
        }
      },
      agent: {
        title: 'Agent 设定',
        description: '定制 Agent 的性格、知识背景与行为逻辑',
        placeholder: '在此输入 Agent 的系统提示词...'
      },
      channels: {
        title: '频道配置',
        description: '配置第三方平台的接入凭证，实现在不同终端的自动化推送与交互',
        feishu: '飞书 (Feishu)',
        dingtalk: '钉钉 (DingTalk)',
        webhook: 'Webhook',
        running: '运行中',
        stopped: '未启用',
        configured: '已激活',
        notConfigured: '未配置',
        endpoint: '接收地址：'
      }
    },
    cron: {
      title: '定时任务',
      add: '新增任务',
      edit: '编辑任务',
      name: '任务名称',
      description: '描述',
      expr: 'Cron 表达式',
      channel: '频道',
      prompt: '触发词',
      status: '状态',
      action: '操作',
      trigger: '立即触发',
      triggerConfirm: '确认立即执行任务 "{name}" 吗？',
      deleteConfirm: '删除任务 "{name}" 后无法恢复，是否继续？',
      window: '活跃时间窗口 (可选)',
      windowStart: '开始',
      windowEnd: '结束',
      noJobs: '暂无定时任务',
      history: '执行历史',
      historyEmpty: '暂无执行记录',
      helper: {
        expr: '例如 "0 9 * * 1-5" 表示工作日 9 点'
      }
    },
    logs: {
      title: '系统日志',
      description: '实时监控系统运行状态与错误日志',
      refresh: '刷新',
      autoRefresh: '自动刷新',
      level: '级别',
      message: '内容',
      time: '时间',
      noLogs: '暂无日志数据'
    },
    setup: {
      title: '欢迎使用 GoPaw',
      description: '请先配置 LLM 提供商以开始使用',
      getStarted: '开始配置',
      configured: '已配置，前往聊天'
    }
  },
  'en-US': {
    common: {
      loading: 'Loading...',
      save: 'Save',
      cancel: 'Cancel',
      delete: 'Delete',
      edit: 'Edit',
      confirm: 'Confirm',
      success: 'Success',
      error: 'Error',
      action: 'Action',
      back: 'Back',
      next: 'Next',
      finish: 'Finish'
    },
    nav: {
      chat: 'Chat',
      settings: 'Settings',
      providers: 'LLM Providers',
      channels: 'Channels',
      agent: 'Agent',
      cron: 'Cron',
      logs: 'System Logs',
      skills: 'Skills'
    },
    chat: {
      title: 'Chat',
      placeholder: 'Type a message...',
      send: 'Send',
      newChat: 'New Chat',
      history: 'History',
      thinking: 'Thinking...',
      welcome: 'Hello! I\'m GoPaw, your AI assistant. How can I help you?',
      deleteConfirm: 'Delete this session and all its messages? This action cannot be undone.'
    },
    settings: {
      title: 'Settings',
      description: 'Manage core system configurations and integrations',
      syncStatus: 'Synced to Cloud',
      modifiedStatus: 'Unsaved Changes',
      markdownTip: 'Markdown supported. Settings will be injected as System Prompt',
      providers: {
        title: 'LLM Providers',
        description: 'Configure and switch LLM providers, supports OpenAI format',
        add: 'Add Provider',
        edit: 'Edit Provider',
        name: 'Name',
        baseURL: 'Base URL',
        apiKey: 'API Key',
        model: 'Model',
        active: 'Active',
        setActive: 'Set Active',
        deleteConfirm: 'Are you sure to delete this provider?',
        noProviders: 'No LLM providers',
        addFirst: 'Add your first LLM provider to get started',
        placeholder: {
          name: 'e.g., OpenAI',
          baseURL: 'https://api.openai.com/v1',
          apiKey: 'sk-...',
          model: 'gpt-4o-mini'
        }
      },
      agent: {
        title: 'Agent Settings',
        description: 'Customize Agent personality, knowledge and behavior',
        placeholder: 'Enter System Prompt here...'
      },
      channels: {
        title: 'Channels',
        description: 'Configure 3rd-party platform credentials for automation',
        feishu: 'Feishu',
        dingtalk: 'DingTalk',
        webhook: 'Webhook',
        running: 'Running',
        stopped: 'Stopped',
        configured: 'Active',
        notConfigured: 'Not Configured',
        endpoint: 'Endpoint: '
      }
    },
    cron: {
      title: 'Cron Jobs',
      add: 'Add Job',
      edit: 'Edit Job',
      name: 'Name',
      description: 'Description',
      expr: 'Cron Expr',
      channel: 'Channel',
      prompt: 'Prompt',
      status: 'Status',
      action: 'Action',
      trigger: 'Trigger Now',
      triggerConfirm: 'Run task "{name}" immediately?',
      deleteConfirm: 'Delete task "{name}"? This action cannot be undone.',
      window: 'Active Time Window (Optional)',
      windowStart: 'Start',
      windowEnd: 'End',
      noJobs: 'No cron jobs',
      history: 'Execution History',
      historyEmpty: 'No execution records',
      helper: {
        expr: 'e.g., "0 9 * * 1-5" for weekdays at 9 AM'
      }
    },
    logs: {
      title: 'System Logs',
      description: 'Real-time monitoring of system status and errors',
      refresh: 'Refresh',
      autoRefresh: 'Auto Refresh',
      level: 'Level',
      message: 'Message',
      time: 'Time',
      noLogs: 'No logs available'
    },
    setup: {
      title: 'Welcome to GoPaw',
      description: 'Please configure an LLM provider to get started',
      getStarted: 'Get Started',
      configured: 'Configured, Go to Chat'
    }
  }
}

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  fallbackLocale: 'en-US',
  messages
})

export default i18n
