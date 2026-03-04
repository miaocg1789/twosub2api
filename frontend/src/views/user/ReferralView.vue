<template>
  <AppLayout>
    <div class="mx-auto max-w-3xl space-y-6">
      <!-- Stats Cards -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div class="card p-6 text-center">
          <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('referral.totalInvitees') }}</p>
          <p class="mt-2 text-3xl font-bold text-gray-900 dark:text-white">
            {{ stats?.total_invited ?? 0 }}
            <span class="text-base font-normal text-gray-400">{{ t('referral.people') }}</span>
          </p>
        </div>
        <div class="card p-6 text-center">
          <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('referral.totalCommission') }}</p>
          <p class="mt-2 text-3xl font-bold text-primary-600 dark:text-primary-400">
            ${{ stats?.total_commission?.toFixed(2) ?? '0.00' }}
          </p>
        </div>
      </div>

      <!-- Referral Link -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('referral.myReferralLink') }}
          </h2>
        </div>
        <div class="p-6">
          <div class="flex items-center gap-3">
            <input
              type="text"
              readonly
              :value="referralLink"
              class="input flex-1 bg-gray-50 dark:bg-dark-800"
              @click="($event.target as HTMLInputElement)?.select()"
            />
            <button
              type="button"
              class="btn btn-primary flex-shrink-0"
              @click="copyLink"
            >
              {{ t('referral.copyLink') }}
            </button>
          </div>
        </div>
      </div>

      <!-- Tabs -->
      <div class="card">
        <div class="border-b border-gray-100 dark:border-dark-700">
          <div class="flex">
            <button
              type="button"
              class="px-6 py-3 text-sm font-medium transition-colors"
              :class="activeTab === 'invitees'
                ? 'border-b-2 border-primary-600 text-primary-600 dark:border-primary-400 dark:text-primary-400'
                : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'"
              @click="activeTab = 'invitees'"
            >
              {{ t('referral.invitees') }}
            </button>
            <button
              type="button"
              class="px-6 py-3 text-sm font-medium transition-colors"
              :class="activeTab === 'commissions'
                ? 'border-b-2 border-primary-600 text-primary-600 dark:border-primary-400 dark:text-primary-400'
                : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'"
              @click="activeTab = 'commissions'"
            >
              {{ t('referral.commissions') }}
            </button>
          </div>
        </div>

        <!-- Invitees Tab -->
        <div v-if="activeTab === 'invitees'" class="p-6">
          <div v-if="invitees.length === 0" class="py-8 text-center text-gray-500 dark:text-gray-400">
            {{ t('referral.noInvitees') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-gray-100 dark:border-dark-700">
                  <th class="pb-3 text-left font-medium text-gray-500 dark:text-gray-400">{{ t('referral.inviteeEmail') }}</th>
                  <th class="pb-3 text-left font-medium text-gray-500 dark:text-gray-400">{{ t('referral.registeredAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="invitee in invitees" :key="invitee.user_id" class="border-b border-gray-50 dark:border-dark-800">
                  <td class="py-3 text-gray-900 dark:text-white">{{ invitee.email }}</td>
                  <td class="py-3 text-gray-500 dark:text-gray-400">{{ formatDate(invitee.created_at) }}</td>
                </tr>
              </tbody>
            </table>
            <!-- Pagination -->
            <div v-if="inviteesTotal > inviteesPageSize" class="mt-4 flex items-center justify-between">
              <span class="text-sm text-gray-500 dark:text-gray-400">
                {{ inviteesTotal }} {{ t('referral.people') }}
              </span>
              <div class="flex gap-2">
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="inviteesPage <= 1"
                  @click="loadInvitees(inviteesPage - 1)"
                >&laquo;</button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="inviteesPage * inviteesPageSize >= inviteesTotal"
                  @click="loadInvitees(inviteesPage + 1)"
                >&raquo;</button>
              </div>
            </div>
          </div>
        </div>

        <!-- Commissions Tab -->
        <div v-if="activeTab === 'commissions'" class="p-6">
          <div v-if="commissions.length === 0" class="py-8 text-center text-gray-500 dark:text-gray-400">
            {{ t('referral.noCommissions') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-gray-100 dark:border-dark-700">
                  <th class="pb-3 text-left font-medium text-gray-500 dark:text-gray-400">{{ t('referral.orderAmount') }}</th>
                  <th class="pb-3 text-left font-medium text-gray-500 dark:text-gray-400">{{ t('referral.commissionRate') }}</th>
                  <th class="pb-3 text-left font-medium text-gray-500 dark:text-gray-400">{{ t('referral.commissionAmount') }}</th>
                  <th class="pb-3 text-left font-medium text-gray-500 dark:text-gray-400">{{ t('referral.commissionTime') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="c in commissions" :key="c.id" class="border-b border-gray-50 dark:border-dark-800">
                  <td class="py-3 text-gray-900 dark:text-white">${{ c.order_amount.toFixed(2) }}</td>
                  <td class="py-3 text-gray-500 dark:text-gray-400">{{ (c.commission_rate * 100).toFixed(1) }}%</td>
                  <td class="py-3 font-medium text-primary-600 dark:text-primary-400">${{ c.commission_amount.toFixed(2) }}</td>
                  <td class="py-3 text-gray-500 dark:text-gray-400">{{ formatDate(c.created_at) }}</td>
                </tr>
              </tbody>
            </table>
            <!-- Pagination -->
            <div v-if="commissionsTotal > commissionsPageSize" class="mt-4 flex items-center justify-between">
              <span class="text-sm text-gray-500 dark:text-gray-400">
                {{ commissionsTotal }} {{ t('referral.commissions') }}
              </span>
              <div class="flex gap-2">
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="commissionsPage <= 1"
                  @click="loadCommissions(commissionsPage - 1)"
                >&laquo;</button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="commissionsPage * commissionsPageSize >= commissionsTotal"
                  @click="loadCommissions(commissionsPage + 1)"
                >&raquo;</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { AppLayout } from '@/components/layout'
import { useAuthStore, useAppStore } from '@/stores'
import { getReferralStats, getInvitees, getCommissions } from '@/api/referral'
import type { ReferralStats, ReferralInvitee, ReferralCommission } from '@/types'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const stats = ref<ReferralStats | null>(null)
const activeTab = ref<'invitees' | 'commissions'>('invitees')

// Invitees
const invitees = ref<ReferralInvitee[]>([])
const inviteesPage = ref(1)
const inviteesPageSize = 20
const inviteesTotal = ref(0)

// Commissions
const commissions = ref<ReferralCommission[]>([])
const commissionsPage = ref(1)
const commissionsPageSize = 20
const commissionsTotal = ref(0)

// Referral link
const referralLink = ref('')

function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function copyLink() {
  navigator.clipboard.writeText(referralLink.value).then(() => {
    appStore.showSuccess(t('referral.linkCopied'))
  })
}

async function loadStats() {
  try {
    stats.value = await getReferralStats()
  } catch (e) {
    console.error('Failed to load referral stats:', e)
  }
}

async function loadInvitees(page = 1) {
  try {
    const result = await getInvitees({ page, page_size: inviteesPageSize })
    invitees.value = result.items || []
    inviteesTotal.value = result.total
    inviteesPage.value = page
  } catch (e) {
    console.error('Failed to load invitees:', e)
  }
}

async function loadCommissions(page = 1) {
  try {
    const result = await getCommissions({ page, page_size: commissionsPageSize })
    commissions.value = result.items || []
    commissionsTotal.value = result.total
    commissionsPage.value = page
  } catch (e) {
    console.error('Failed to load commissions:', e)
  }
}

onMounted(() => {
  // Build referral link
  const userId = authStore.user?.id
  if (userId) {
    referralLink.value = `${window.location.origin}/register?aff=${userId}`
  }

  loadStats()
  loadInvitees()
  loadCommissions()
})
</script>
