<template>
  <AppLayout>
    <div class="px-2 md:px-6 py-6 max-w-[1000px] mx-auto">
      <div class="mb-6">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('installGuide.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('installGuide.subtitle') }}</p>
      </div>

      <div class="space-y-3">
        <div
          v-for="tool in tools"
          :key="tool.key"
          class="rounded-xl border border-gray-200 dark:border-dark-700 bg-white dark:bg-dark-800/50 overflow-hidden transition-all"
        >
          <!-- Header (clickable) -->
          <button
            @click="toggle(tool.key)"
            class="w-full flex items-center justify-between px-5 py-4 text-left hover:bg-gray-50 dark:hover:bg-dark-700/30 transition-colors"
          >
            <div class="flex items-center gap-3">
              <div class="flex items-center justify-center w-9 h-9 rounded-lg" :class="tool.iconBg">
                <component :is="tool.iconComponent" class="w-5 h-5" :class="tool.iconColor" />
              </div>
              <div>
                <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ tool.name }}</span>
                <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{{ tool.desc }}</p>
              </div>
            </div>
            <svg
              class="w-5 h-5 text-gray-400 dark:text-gray-500 transition-transform duration-200"
              :class="{ 'rotate-180': expanded[tool.key] }"
              fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
            >
              <path d="m6 9 6 6 6-6" />
            </svg>
          </button>

          <!-- Expandable Content -->
          <transition name="accordion">
            <div v-if="expanded[tool.key]" class="border-t border-gray-100 dark:border-dark-700">
              <!-- OS Tabs -->
              <div class="flex border-b border-gray-100 dark:border-dark-700 bg-gray-50/50 dark:bg-dark-800/80">
                <button
                  v-for="os in osList"
                  :key="os.key"
                  @click="activeOs[tool.key] = os.key"
                  class="flex items-center gap-1.5 px-4 py-2.5 text-xs font-medium transition-all border-b-2 -mb-px"
                  :class="activeOs[tool.key] === os.key
                    ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                    : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'"
                >
                  <component :is="os.icon" class="w-3.5 h-3.5" />
                  {{ os.label }}
                </button>
              </div>

              <!-- Install Steps -->
              <div class="px-5 py-4 space-y-4">
                <div
                  v-for="(step, idx) in getSteps(tool.key, activeOs[tool.key])"
                  :key="idx"
                  class="space-y-2"
                >
                  <div class="flex items-start gap-2">
                    <span class="flex-shrink-0 w-5 h-5 rounded-full bg-primary-100 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400 text-[11px] font-bold flex items-center justify-center mt-0.5">{{ idx + 1 }}</span>
                    <p class="text-sm text-gray-700 dark:text-gray-300">{{ step.title }}</p>
                  </div>
                  <div v-if="step.code" class="ml-7 relative group">
                    <pre class="text-xs font-mono bg-gray-900 dark:bg-dark-950 text-gray-100 rounded-lg px-4 py-3 overflow-x-auto"><code>{{ step.code }}</code></pre>
                    <button
                      @click="copyCode(step.code)"
                      class="absolute top-2 right-2 p-1.5 rounded-md bg-gray-700/50 hover:bg-gray-600 text-gray-300 hover:text-white transition-all opacity-0 group-hover:opacity-100"
                      :title="t('installGuide.copyCommand')"
                    >
                      <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
                        <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
                      </svg>
                    </button>
                  </div>
                  <p v-if="step.note" class="ml-7 text-xs text-gray-500 dark:text-gray-400 leading-relaxed">{{ step.note }}</p>
                </div>
              </div>
            </div>
          </transition>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { reactive, h } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'

const { t } = useI18n()

// Expanded state
const expanded = reactive<Record<string, boolean>>({
  claudeCode: false,
  codex: false,
  geminiCli: false,
  githubCopilot: false,
  cursor: false,
  cline: false,
  openCode: false,
  openClaw: false,
})
const activeOs = reactive<Record<string, string>>({
  claudeCode: 'windows',
  codex: 'windows',
  geminiCli: 'windows',
  githubCopilot: 'windows',
  cursor: 'windows',
  cline: 'windows',
  openCode: 'windows',
  openClaw: 'windows',
})

function toggle(key: string) {
  expanded[key] = !expanded[key]
}

// OS list
const WindowsIcon = {
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'currentColor' }, [
    h('path', { d: 'M3 12V6.75l8-1.25V12H3zm0 .5h8v6.5l-8-1.25V12.5zM11.5 5.38l9.5-1.63V12h-9.5V5.38zM11.5 12.5H21v7.25l-9.5-1.63V12.5z' })
  ])
}
const MacIcon = {
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'currentColor' }, [
    h('path', { d: 'M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z' })
  ])
}
const LinuxIcon = {
  render: () => h('svg', { viewBox: '0 0 24 24', fill: 'currentColor' }, [
    h('path', { d: 'M12.504 0c-.155 0-.311.015-.466.046-2.72.557-3.39 4.327-3.686 6.15-.136.834-.18 1.39-.18 1.804 0 .94.218 1.678.622 2.197-.94.96-1.678 2.2-1.678 3.803 0 1.202.466 2.218 1.168 2.963-.232.466-.388.994-.388 1.584 0 1.946 1.584 3.528 3.528 3.528.7 0 1.35-.204 1.898-.558.548.354 1.198.558 1.898.558 1.944 0 3.527-1.582 3.527-3.528 0-.59-.155-1.118-.388-1.584.703-.745 1.168-1.76 1.168-2.963 0-1.604-.738-2.843-1.678-3.803.404-.519.622-1.257.622-2.197 0-.414-.044-.97-.18-1.804-.296-1.823-.965-5.593-3.686-6.15A3.44 3.44 0 0012.504 0z' })
  ])
}

const osList = [
  { key: 'windows', label: 'Windows', icon: WindowsIcon },
  { key: 'macos', label: 'macOS', icon: MacIcon },
  { key: 'linux', label: 'Linux', icon: LinuxIcon },
]

// Tool icon component
const TerminalIcon = {
  render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
    h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z' })
  ])
}
const CodeIcon = {
  render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
    h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5' })
  ])
}
const PuzzleIcon = {
  render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
    h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M14.25 6.087c0-.355.186-.676.401-.959.221-.29.349-.634.349-1.003 0-1.036-1.007-1.875-2.25-1.875s-2.25.84-2.25 1.875c0 .369.128.713.349 1.003.215.283.401.604.401.959v0a.64.64 0 01-.657.643 48.39 48.39 0 01-4.163-.3c.186 1.613.466 3.2.836 4.755a48.345 48.345 0 01-4.163.3.64.64 0 00-.657.643v0c0 .355.186.676.401.959.221.29.349.634.349 1.003 0 1.035-1.007 1.875-2.25 1.875S.75 14.786.75 13.75c0-.369.128-.713.349-1.003.215-.283.401-.604.401-.959v0a.64.64 0 00-.657-.643A49.412 49.412 0 004.5 10.5' })
  ])
}
const CursorIcon = {
  render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
    h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M15.042 21.672L13.684 16.6m0 0l-2.51 2.225.569-9.47 5.227 7.917-3.286-.672zM12 2.25V4.5m5.834.166l-1.591 1.591M20.25 10.5H18M7.757 14.743l-1.59 1.59M6 10.5H3.75m4.007-4.243l-1.59-1.59' })
  ])
}

// Tools configuration
const tools = [
  {
    key: 'claudeCode',
    name: 'Claude Code',
    desc: t('installGuide.claudeCodeDesc'),
    iconBg: 'bg-orange-100 dark:bg-orange-900/30',
    iconColor: 'text-orange-600 dark:text-orange-400',
    iconComponent: TerminalIcon,
  },
  {
    key: 'codex',
    name: 'Codex',
    desc: t('installGuide.codexDesc'),
    iconBg: 'bg-emerald-100 dark:bg-emerald-900/30',
    iconColor: 'text-emerald-600 dark:text-emerald-400',
    iconComponent: TerminalIcon,
  },
  {
    key: 'geminiCli',
    name: 'Gemini CLI',
    desc: t('installGuide.geminiCliDesc'),
    iconBg: 'bg-blue-100 dark:bg-blue-900/30',
    iconColor: 'text-blue-600 dark:text-blue-400',
    iconComponent: TerminalIcon,
  },
  {
    key: 'githubCopilot',
    name: 'GitHub Copilot',
    desc: t('installGuide.githubCopilotDesc'),
    iconBg: 'bg-gray-100 dark:bg-gray-800',
    iconColor: 'text-gray-700 dark:text-gray-300',
    iconComponent: CodeIcon,
  },
  {
    key: 'cursor',
    name: 'Cursor',
    desc: t('installGuide.cursorDesc'),
    iconBg: 'bg-purple-100 dark:bg-purple-900/30',
    iconColor: 'text-purple-600 dark:text-purple-400',
    iconComponent: CursorIcon,
  },
  {
    key: 'cline',
    name: 'Cline',
    desc: t('installGuide.clineDesc'),
    iconBg: 'bg-teal-100 dark:bg-teal-900/30',
    iconColor: 'text-teal-600 dark:text-teal-400',
    iconComponent: PuzzleIcon,
  },
  {
    key: 'openCode',
    name: 'OpenCode',
    desc: t('installGuide.openCodeDesc'),
    iconBg: 'bg-indigo-100 dark:bg-indigo-900/30',
    iconColor: 'text-indigo-600 dark:text-indigo-400',
    iconComponent: TerminalIcon,
  },
  {
    key: 'openClaw',
    name: 'OpenClaw',
    desc: t('installGuide.openClawDesc'),
    iconBg: 'bg-pink-100 dark:bg-pink-900/30',
    iconColor: 'text-pink-600 dark:text-pink-400',
    iconComponent: TerminalIcon,
  },
]

interface Step {
  title: string
  code?: string
  note?: string
}

function getSteps(toolKey: string, os: string): Step[] {
  const data: Record<string, Record<string, Step[]>> = {
    claudeCode: {
      windows: [
        { title: t('installGuide.steps.installNodejs'), code: 'winget install OpenJS.NodeJS.LTS', note: t('installGuide.steps.installNodejsNote') },
        { title: t('installGuide.steps.installClaudeCode'), code: 'npm install -g @anthropic-ai/claude-code' },
        { title: t('installGuide.steps.configureEnv'), code: 'set ANTHROPIC_BASE_URL=https://your-api-domain.com\nset ANTHROPIC_API_KEY=sk-your-api-key' },
        { title: t('installGuide.steps.run'), code: 'claude', note: t('installGuide.steps.claudeCodeNote') },
      ],
      macos: [
        { title: t('installGuide.steps.installNodejs'), code: 'brew install node', note: t('installGuide.steps.installNodejsNote') },
        { title: t('installGuide.steps.installClaudeCode'), code: 'npm install -g @anthropic-ai/claude-code' },
        { title: t('installGuide.steps.configureEnv'), code: 'export ANTHROPIC_BASE_URL=https://your-api-domain.com\nexport ANTHROPIC_API_KEY=sk-your-api-key' },
        { title: t('installGuide.steps.run'), code: 'claude', note: t('installGuide.steps.claudeCodeNote') },
      ],
      linux: [
        { title: t('installGuide.steps.installNodejs'), code: 'curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -\nsudo apt-get install -y nodejs', note: t('installGuide.steps.installNodejsNote') },
        { title: t('installGuide.steps.installClaudeCode'), code: 'npm install -g @anthropic-ai/claude-code' },
        { title: t('installGuide.steps.configureEnv'), code: 'export ANTHROPIC_BASE_URL=https://your-api-domain.com\nexport ANTHROPIC_API_KEY=sk-your-api-key' },
        { title: t('installGuide.steps.run'), code: 'claude', note: t('installGuide.steps.claudeCodeNote') },
      ],
    },
    codex: {
      windows: [
        { title: t('installGuide.steps.installNodejs'), code: 'winget install OpenJS.NodeJS.LTS' },
        { title: t('installGuide.steps.installTool', { name: 'Codex' }), code: 'npm install -g @openai/codex' },
        { title: t('installGuide.steps.configureEnv'), code: 'set OPENAI_BASE_URL=https://your-api-domain.com/v1\nset OPENAI_API_KEY=sk-your-api-key' },
        { title: t('installGuide.steps.run'), code: 'codex', note: t('installGuide.steps.codexNote') },
      ],
      macos: [
        { title: t('installGuide.steps.installNodejs'), code: 'brew install node' },
        { title: t('installGuide.steps.installTool', { name: 'Codex' }), code: 'npm install -g @openai/codex' },
        { title: t('installGuide.steps.configureEnv'), code: 'export OPENAI_BASE_URL=https://your-api-domain.com/v1\nexport OPENAI_API_KEY=sk-your-api-key' },
        { title: t('installGuide.steps.run'), code: 'codex', note: t('installGuide.steps.codexNote') },
      ],
      linux: [
        { title: t('installGuide.steps.installNodejs'), code: 'curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -\nsudo apt-get install -y nodejs' },
        { title: t('installGuide.steps.installTool', { name: 'Codex' }), code: 'npm install -g @openai/codex' },
        { title: t('installGuide.steps.configureEnv'), code: 'export OPENAI_BASE_URL=https://your-api-domain.com/v1\nexport OPENAI_API_KEY=sk-your-api-key' },
        { title: t('installGuide.steps.run'), code: 'codex', note: t('installGuide.steps.codexNote') },
      ],
    },
    geminiCli: {
      windows: [
        { title: t('installGuide.steps.installNodejs'), code: 'winget install OpenJS.NodeJS.LTS' },
        { title: t('installGuide.steps.installTool', { name: 'Gemini CLI' }), code: 'npm install -g @anthropic-ai/gemini-cli', note: t('installGuide.steps.geminiCliNote') },
        { title: t('installGuide.steps.configureEnv'), code: 'set GEMINI_API_KEY=your-api-key' },
        { title: t('installGuide.steps.run'), code: 'gemini' },
      ],
      macos: [
        { title: t('installGuide.steps.installNodejs'), code: 'brew install node' },
        { title: t('installGuide.steps.installTool', { name: 'Gemini CLI' }), code: 'npm install -g @anthropic-ai/gemini-cli' },
        { title: t('installGuide.steps.configureEnv'), code: 'export GEMINI_API_KEY=your-api-key' },
        { title: t('installGuide.steps.run'), code: 'gemini' },
      ],
      linux: [
        { title: t('installGuide.steps.installNodejs'), code: 'curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -\nsudo apt-get install -y nodejs' },
        { title: t('installGuide.steps.installTool', { name: 'Gemini CLI' }), code: 'npm install -g @anthropic-ai/gemini-cli' },
        { title: t('installGuide.steps.configureEnv'), code: 'export GEMINI_API_KEY=your-api-key' },
        { title: t('installGuide.steps.run'), code: 'gemini' },
      ],
    },
    githubCopilot: {
      windows: [
        { title: t('installGuide.steps.installVscode'), code: 'winget install Microsoft.VisualStudioCode', note: t('installGuide.steps.vscodeNote') },
        { title: t('installGuide.steps.installExtension', { name: 'GitHub Copilot' }), code: 'code --install-extension GitHub.copilot\ncode --install-extension GitHub.copilot-chat' },
        { title: t('installGuide.steps.configureGithubCopilot'), note: t('installGuide.steps.githubCopilotNote') },
      ],
      macos: [
        { title: t('installGuide.steps.installVscode'), code: 'brew install --cask visual-studio-code', note: t('installGuide.steps.vscodeNote') },
        { title: t('installGuide.steps.installExtension', { name: 'GitHub Copilot' }), code: 'code --install-extension GitHub.copilot\ncode --install-extension GitHub.copilot-chat' },
        { title: t('installGuide.steps.configureGithubCopilot'), note: t('installGuide.steps.githubCopilotNote') },
      ],
      linux: [
        { title: t('installGuide.steps.installVscode'), code: 'sudo snap install code --classic', note: t('installGuide.steps.vscodeAlternative') },
        { title: t('installGuide.steps.installExtension', { name: 'GitHub Copilot' }), code: 'code --install-extension GitHub.copilot\ncode --install-extension GitHub.copilot-chat' },
        { title: t('installGuide.steps.configureGithubCopilot'), note: t('installGuide.steps.githubCopilotNote') },
      ],
    },
    cursor: {
      windows: [
        { title: t('installGuide.steps.downloadCursor'), code: 'winget install Anysphere.Cursor', note: t('installGuide.steps.cursorDownloadNote') },
        { title: t('installGuide.steps.configureCursor'), note: t('installGuide.steps.cursorConfigNote') },
        { title: t('installGuide.steps.configureCursorModel'), code: '{\n  "openai.apiBaseUrl": "https://your-api-domain.com/v1",\n  "openai.apiKey": "sk-your-api-key"\n}', note: t('installGuide.steps.cursorModelNote') },
      ],
      macos: [
        { title: t('installGuide.steps.downloadCursor'), code: 'brew install --cask cursor', note: t('installGuide.steps.cursorDownloadNote') },
        { title: t('installGuide.steps.configureCursor'), note: t('installGuide.steps.cursorConfigNote') },
        { title: t('installGuide.steps.configureCursorModel'), code: '{\n  "openai.apiBaseUrl": "https://your-api-domain.com/v1",\n  "openai.apiKey": "sk-your-api-key"\n}', note: t('installGuide.steps.cursorModelNote') },
      ],
      linux: [
        { title: t('installGuide.steps.downloadCursor'), note: t('installGuide.steps.cursorLinuxNote') },
        { title: t('installGuide.steps.installCursorLinux'), code: 'chmod +x cursor-*.AppImage\n./cursor-*.AppImage' },
        { title: t('installGuide.steps.configureCursor'), note: t('installGuide.steps.cursorConfigNote') },
        { title: t('installGuide.steps.configureCursorModel'), code: '{\n  "openai.apiBaseUrl": "https://your-api-domain.com/v1",\n  "openai.apiKey": "sk-your-api-key"\n}', note: t('installGuide.steps.cursorModelNote') },
      ],
    },
    cline: {
      windows: [
        { title: t('installGuide.steps.installVscode'), code: 'winget install Microsoft.VisualStudioCode' },
        { title: t('installGuide.steps.installExtension', { name: 'Cline' }), code: 'code --install-extension saoudrizwan.claude-dev' },
        { title: t('installGuide.steps.configureCline'), note: t('installGuide.steps.clineConfigNote') },
      ],
      macos: [
        { title: t('installGuide.steps.installVscode'), code: 'brew install --cask visual-studio-code' },
        { title: t('installGuide.steps.installExtension', { name: 'Cline' }), code: 'code --install-extension saoudrizwan.claude-dev' },
        { title: t('installGuide.steps.configureCline'), note: t('installGuide.steps.clineConfigNote') },
      ],
      linux: [
        { title: t('installGuide.steps.installVscode'), code: 'sudo snap install code --classic' },
        { title: t('installGuide.steps.installExtension', { name: 'Cline' }), code: 'code --install-extension saoudrizwan.claude-dev' },
        { title: t('installGuide.steps.configureCline'), note: t('installGuide.steps.clineConfigNote') },
      ],
    },
    openCode: {
      windows: [
        { title: t('installGuide.steps.installGo'), code: 'winget install GoLang.Go', note: t('installGuide.steps.goNote') },
        { title: t('installGuide.steps.installTool', { name: 'OpenCode' }), code: 'go install github.com/opencode-ai/opencode@latest' },
        { title: t('installGuide.steps.configureEnv'), code: 'set OPENAI_BASE_URL=https://your-api-domain.com/v1\nset OPENAI_API_KEY=sk-your-api-key' },
        { title: t('installGuide.steps.run'), code: 'opencode' },
      ],
      macos: [
        { title: t('installGuide.steps.installGo'), code: 'brew install go' },
        { title: t('installGuide.steps.installTool', { name: 'OpenCode' }), code: 'go install github.com/opencode-ai/opencode@latest' },
        { title: t('installGuide.steps.configureEnv'), code: 'export OPENAI_BASE_URL=https://your-api-domain.com/v1\nexport OPENAI_API_KEY=sk-your-api-key' },
        { title: t('installGuide.steps.run'), code: 'opencode' },
      ],
      linux: [
        { title: t('installGuide.steps.installGo'), code: 'sudo apt-get install -y golang-go', note: t('installGuide.steps.goNote') },
        { title: t('installGuide.steps.installTool', { name: 'OpenCode' }), code: 'go install github.com/opencode-ai/opencode@latest' },
        { title: t('installGuide.steps.configureEnv'), code: 'export OPENAI_BASE_URL=https://your-api-domain.com/v1\nexport OPENAI_API_KEY=sk-your-api-key' },
        { title: t('installGuide.steps.run'), code: 'opencode' },
      ],
    },
    openClaw: {
      windows: [
        { title: t('installGuide.steps.installPython'), code: 'winget install Python.Python.3.12', note: t('installGuide.steps.pythonNote') },
        { title: t('installGuide.steps.installTool', { name: 'OpenClaw' }), code: 'pip install openclaw' },
        { title: t('installGuide.steps.configureEnv'), code: 'set OPENAI_BASE_URL=https://your-api-domain.com/v1\nset OPENAI_API_KEY=sk-your-api-key' },
        { title: t('installGuide.steps.run'), code: 'openclaw' },
      ],
      macos: [
        { title: t('installGuide.steps.installPython'), code: 'brew install python@3.12' },
        { title: t('installGuide.steps.installTool', { name: 'OpenClaw' }), code: 'pip3 install openclaw' },
        { title: t('installGuide.steps.configureEnv'), code: 'export OPENAI_BASE_URL=https://your-api-domain.com/v1\nexport OPENAI_API_KEY=sk-your-api-key' },
        { title: t('installGuide.steps.run'), code: 'openclaw' },
      ],
      linux: [
        { title: t('installGuide.steps.installPython'), code: 'sudo apt-get install -y python3 python3-pip' },
        { title: t('installGuide.steps.installTool', { name: 'OpenClaw' }), code: 'pip3 install openclaw' },
        { title: t('installGuide.steps.configureEnv'), code: 'export OPENAI_BASE_URL=https://your-api-domain.com/v1\nexport OPENAI_API_KEY=sk-your-api-key' },
        { title: t('installGuide.steps.run'), code: 'openclaw' },
      ],
    },
  }
  return data[toolKey]?.[os] || []
}

async function copyCode(code: string) {
  try {
    await navigator.clipboard.writeText(code)
  } catch {
    const ta = document.createElement('textarea')
    ta.value = code
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
  }
}
</script>

<style scoped>
.accordion-enter-active {
  transition: all 0.25s ease-out;
  overflow: hidden;
}
.accordion-leave-active {
  transition: all 0.2s ease-in;
  overflow: hidden;
}
.accordion-enter-from,
.accordion-leave-to {
  opacity: 0;
  max-height: 0;
}
.accordion-enter-to,
.accordion-leave-from {
  opacity: 1;
  max-height: 800px;
}
</style>
