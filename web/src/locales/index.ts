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
      cron: '定时任务'
    },
    chat: {
      title: '聊天',
      placeholder: '输入消息...',
      send: '发送',
      newChat: '新对话',
      history: '历史会话',
      thinking: '思考中...',
      welcome: '你好！我是 GoPaw，你的 AI 助理。有什么可以帮你的吗？'
    },
    settings: {
      title: '设置',
      providers: {
        title: 'LLM 提供商',
        add: '添加提供商',
        edit: '编辑提供商',
        name: '名称',
        baseURL: 'API 地址',
        apiKey: 'API Key',
        model: '模型',
        active: '活跃',
        setActive: '设为活跃',
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
        description: '编辑 Agent 的角色设定和系统提示',
        placeholder: '输入 Agent 设定...'
      },
      channels: {
        title: '频道配置',
        feishu: '飞书',
        dingtalk: '钉钉',
        webhook: 'Webhook',
        health: '健康状态',
        running: '运行中',
        stopped: '已停止'
      }
    },
    cron: {
      title: '定时任务',
      add: '新增任务',
      name: '任务名称',
      expr: 'Cron 表达式',
      channel: '频道',
      prompt: '触发词',
      status: '状态',
      action: '操作',
      trigger: '立即触发',
      window: '活跃窗口',
      noJobs: '暂无定时任务',
      helper: {
        expr: '例如 "0 9 * * 1-5" 表示工作日 9 点'
      }
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
      cron: 'Cron'
    },
    chat: {
      title: 'Chat',
      placeholder: 'Type a message...',
      send: 'Send',
      newChat: 'New Chat',
      history: 'History',
      thinking: 'Thinking...',
      welcome: 'Hello! I\'m GoPaw, your AI assistant. How can I help you?'
    },
    settings: {
      title: 'Settings',
      providers: {
        title: 'LLM Providers',
        add: 'Add Provider',
        edit: 'Edit Provider',
        name: 'Name',
        baseURL: 'Base URL',
        apiKey: 'API Key',
        model: 'Model',
        active: 'Active',
        setActive: 'Set Active',
        noProviders: 'No LLM providers',
        addFirst: 'Please add your first LLM provider to get started',
        placeholder: {
          name: 'e.g., OpenAI',
          baseURL: 'https://api.openai.com/v1',
          apiKey: 'sk-...',
          model: 'gpt-4o-mini'
        }
      },
      agent: {
        title: 'Agent Settings',
        description: 'Edit Agent role and system prompt',
        placeholder: 'Enter agent settings...'
      },
      channels: {
        title: 'Channel Settings',
        feishu: 'Feishu',
        dingtalk: 'DingTalk',
        webhook: 'Webhook',
        health: 'Health',
        running: 'Running',
        stopped: 'Stopped'
      }
    },
    cron: {
      title: 'Cron Jobs',
      add: 'Add Job',
      name: 'Name',
      expr: 'Cron Expr',
      channel: 'Channel',
      prompt: 'Prompt',
      status: 'Status',
      action: 'Action',
      trigger: 'Trigger Now',
      window: 'Active Window',
      noJobs: 'No cron jobs',
      helper: {
        expr: 'e.g., "0 9 * * 1-5" for weekdays at 9 AM'
      }
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
