<template>
  <AppLayout>
    <div class="mx-auto max-w-2xl space-y-6">
      <!-- Current Balance Card -->
      <div class="card overflow-hidden">
        <div class="bg-gradient-to-br from-primary-500 to-primary-600 px-6 py-8 text-center">
          <div
            class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-white/20 backdrop-blur-sm"
          >
            <Icon name="creditCard" size="xl" class="text-white" />
          </div>
          <p class="text-sm font-medium text-primary-100">{{ t('wallet.currentBalance') }}</p>
          <p class="mt-2 text-4xl font-bold text-white">
            ${{ user?.balance?.toFixed(2) || '0.00' }}
          </p>
          <p class="mt-2 text-sm text-primary-100">
            {{ t('wallet.concurrency') }}: {{ user?.concurrency || 0 }} {{ t('wallet.requests') }}
          </p>
        </div>
      </div>

      <!-- Online Top-up Section -->
      <div v-if="paymentConfig?.enabled" class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('wallet.topUp') }}
          </h2>
        </div>
        <div class="space-y-5 p-6">
          <!-- Preset Amounts -->
          <div v-if="paymentConfig && paymentConfig.preset_amounts.length > 0">
            <label class="input-label mb-2">{{ t('wallet.selectAmount') }}</label>
            <div class="grid grid-cols-3 gap-3">
              <button
                v-for="amount in paymentConfig.preset_amounts"
                :key="amount"
                type="button"
                :class="[
                  'rounded-xl border-2 px-4 py-3 text-center font-semibold transition-all',
                  selectedAmount === amount
                    ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/20 dark:text-primary-300'
                    : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-dark-500'
                ]"
                @click="selectPresetAmount(amount)"
              >
                {{ paymentConfig.currency === 'CNY' ? '\u00a5' : '$' }}{{ amount }}
              </button>
            </div>
          </div>

          <!-- Custom Amount -->
          <div>
            <label class="input-label mb-1">{{ t('wallet.customAmount') }}</label>
            <div class="relative mt-1">
              <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
                <span class="text-gray-400 dark:text-dark-500">{{ paymentConfig?.currency === 'CNY' ? '\u00a5' : '$' }}</span>
              </div>
              <input
                v-model.number="customAmount"
                type="number"
                :min="paymentConfig?.min_amount || 1"
                :max="paymentConfig?.max_amount || 10000"
                step="0.01"
                class="input py-3 pl-10 text-lg"
                :placeholder="t('wallet.enterAmount')"
                @input="onCustomAmountInput"
              />
            </div>
            <p class="input-hint">
              {{ t('wallet.amountRange', {
                min: paymentConfig?.min_amount || 1,
                max: paymentConfig?.max_amount || 10000,
                currency: paymentConfig?.currency === 'CNY' ? '\u00a5' : '$'
              }) }}
            </p>
            <!-- Exchange Rate Info -->
            <p v-if="paymentConfig && paymentConfig.exchange_rate !== 1 && effectiveAmount > 0" class="mt-1 text-sm text-primary-600 dark:text-primary-400">
              {{ t('wallet.creditPreview', { credit: (effectiveAmount * paymentConfig.exchange_rate).toFixed(2) }) }}
            </p>
          </div>

          <!-- Payment Method -->
          <div v-if="availableMethods.length > 0">
            <label class="input-label mb-2">{{ t('wallet.paymentMethod') }}</label>
            <div class="grid grid-cols-2 gap-3">
              <button
                v-for="method in availableMethods"
                :key="method.value"
                type="button"
                :class="[
                  'flex items-center gap-3 rounded-xl border-2 px-4 py-3 transition-all',
                  selectedMethod === method.value
                    ? 'border-primary-500 bg-primary-50 dark:border-primary-400 dark:bg-primary-900/20'
                    : 'border-gray-200 bg-white hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:hover:border-dark-500'
                ]"
                @click="selectedMethod = method.value"
              >
                <!-- Alipay icon -->
                <svg v-if="isAlipayMethod(method.value)" class="h-5 w-5 flex-shrink-0" viewBox="0 0 24 24">
                  <circle cx="12" cy="12" r="10" fill="#1677FF"/>
                  <path fill="white" d="M18.36 16.83c-1.05-.46-3.2-1.38-5.27-2.39.9-1.24 1.62-2.66 2.1-4.2H12V8.86h3.6V7.85H12V5.47a.41.41 0 00-.41-.41h-1.33v2.79H7v1.01h3.22v1.38H7.08v1h6.88c-.38 1.2-.93 2.3-1.62 3.2-1.68-.71-3.5-1.31-5.1-1.31-2.48 0-4.08 1.18-4.08 3.05 0 1.96 1.54 3.38 4.59 3.38 2.24 0 4.29-.97 5.87-2.52 1.81.93 4.62 2.2 5.2 2.46.2.1.4 0 .51-.18l.5-1.05c.1-.17 0-.4-.21-.5z"/>
                  <path fill="white" d="M8.5 17.25c-2.22 0-3.13-.88-3.13-1.98 0-1.06.87-1.92 2.6-1.92 1.42 0 3.03.57 4.6 1.32-1.3 1.61-2.69 2.58-4.07 2.58z"/>
                </svg>
                <!-- WeChat icon -->
                <svg v-else-if="isWechatMethod(method.value)" class="h-5 w-5 flex-shrink-0 text-[#07C160]" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M9.5 4C5.36 4 2 6.69 2 10c0 1.89 1.08 3.56 2.78 4.66l-.7 2.1 2.6-1.37C7.6 15.78 8.53 16 9.5 16c.26 0 .52-.01.77-.04A5.46 5.46 0 0010 14.5c0-3.04 2.69-5.5 6-5.5.34 0 .67.03 1 .07C16.36 6.08 13.22 4 9.5 4zM6.75 7.25a1 1 0 110 2 1 1 0 010-2zm5.5 0a1 1 0 110 2 1 1 0 010-2zM16 10c-2.76 0-5 2.01-5 4.5S13.24 19 16 19c.72 0 1.4-.14 2.02-.38L20.5 20l-.5-1.88C21.23 17.12 22 15.88 22 14.5c0-2.49-2.69-4.5-6-4.5zm-2 3a.75.75 0 110 1.5.75.75 0 010-1.5zm4 0a.75.75 0 110 1.5.75.75 0 010-1.5z"/>
                </svg>
                <!-- Default credit card icon -->
                <Icon v-else name="creditCard" size="md" class="flex-shrink-0 text-gray-400" />
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ method.label }}</span>
              </button>
            </div>
          </div>

          <!-- Submit Button -->
          <button
            type="button"
            :disabled="!canSubmitPayment || creatingOrder"
            class="btn btn-primary w-full py-3"
            @click="handleCreateOrder"
          >
            <svg
              v-if="creatingOrder"
              class="-ml-1 mr-2 h-5 w-5 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            {{ creatingOrder ? t('wallet.creating') : t('wallet.payNow') }}
          </button>
        </div>
      </div>

      <!-- Payment QR Code / URL Modal -->
      <teleport to="body">
        <transition name="modal">
          <div v-if="showPaymentDialog" class="fixed inset-0 z-50 flex items-center justify-center">
            <!-- Backdrop -->
            <div class="absolute inset-0 bg-black/50 backdrop-blur-sm" @click="closePaymentDialog"></div>
            <!-- Modal Content -->
            <div class="relative mx-4 w-full max-w-sm rounded-2xl bg-white p-6 shadow-2xl dark:bg-dark-800">
              <div class="flex items-start justify-between">
                <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('wallet.completePayment') }}</h3>
                <button type="button" class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300" @click="closePaymentDialog">
                  <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>

              <div class="mt-4 text-center">
              <!-- Payment Success State -->
              <div v-if="paymentPaidInDialog" class="space-y-4 py-4">
                <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-900/30">
                  <svg class="h-8 w-8 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                </div>
                <p class="text-lg font-semibold text-emerald-600 dark:text-emerald-400">{{ t('wallet.paymentSuccessTitle') }}</p>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('wallet.paymentSuccessMessage') }}</p>
                <button type="button" class="btn btn-primary mt-2" @click="closePaymentDialog">{{ t('common.close') }}</button>
              </div>

              <template v-else>
                <!-- QR Code -->
                <div v-if="qrCodeDataUrl" class="space-y-3">
                  <p class="text-sm text-gray-600 dark:text-gray-400">{{ t('wallet.scanQrCode') }}</p>
                  <div class="mx-auto flex h-48 w-48 items-center justify-center rounded-xl border bg-white p-2">
                    <img :src="qrCodeDataUrl" alt="QR Code" class="h-full w-full object-contain" />
                  </div>
                </div>

                <!-- Payment URL -->
                <div v-else-if="paymentResult?.payment_url" class="space-y-3">
                  <p class="text-sm text-gray-600 dark:text-gray-400">{{ t('wallet.redirecting') }}</p>
                  <a
                    :href="paymentResult.payment_url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="btn btn-primary"
                  >
                    {{ t('wallet.goToPay') }}
                  </a>
                </div>

                <!-- Countdown & Status -->
                <div class="mt-4 flex items-center justify-center gap-2 text-sm text-gray-500 dark:text-gray-400">
                  <svg class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  {{ t('wallet.waitingPayment') }}
                  <span v-if="pollCountdown > 0">({{ pollCountdown }}s)</span>
                </div>

                <!-- Manual Check Button -->
                <button
                  type="button"
                  class="mt-3 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
                  :disabled="manualChecking"
                  @click="manualCheckStatus"
                >
                  {{ manualChecking ? t('wallet.checking') : t('wallet.alreadyPaid') }}
                </button>
              </template>
              </div>
            </div>
          </div>
        </transition>
      </teleport>

      <!-- Payment Success Message -->
      <transition name="fade">
        <div
          v-if="paymentSuccess"
          class="card border-emerald-200 bg-emerald-50 dark:border-emerald-800/50 dark:bg-emerald-900/20"
        >
          <div class="p-6">
            <div class="flex items-start gap-4">
              <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-emerald-100 dark:bg-emerald-900/30">
                <Icon name="checkCircle" size="md" class="text-emerald-600 dark:text-emerald-400" />
              </div>
              <div class="flex-1">
                <h3 class="text-sm font-semibold text-emerald-800 dark:text-emerald-300">
                  {{ t('wallet.paymentSuccessTitle') }}
                </h3>
                <p class="mt-2 text-sm text-emerald-700 dark:text-emerald-400">
                  {{ t('wallet.paymentSuccessMessage') }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <!-- Redeem Code Section -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('redeem.title') }}
          </h2>
        </div>
        <div class="p-6">
          <form @submit.prevent="handleRedeem" class="space-y-5">
            <div>
              <label for="code" class="input-label">
                {{ t('redeem.redeemCodeLabel') }}
              </label>
              <div class="relative mt-1">
                <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
                  <Icon name="gift" size="md" class="text-gray-400 dark:text-dark-500" />
                </div>
                <input
                  id="code"
                  v-model="redeemCode"
                  type="text"
                  required
                  :placeholder="t('redeem.redeemCodePlaceholder')"
                  :disabled="submitting"
                  class="input py-3 pl-12 text-lg"
                />
              </div>
              <p class="input-hint">{{ t('redeem.redeemCodeHint') }}</p>
            </div>

            <button
              type="submit"
              :disabled="!redeemCode || submitting"
              class="btn btn-primary w-full py-3"
            >
              <svg
                v-if="submitting"
                class="-ml-1 mr-2 h-5 w-5 animate-spin"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <Icon v-else name="checkCircle" size="md" class="mr-2" />
              {{ submitting ? t('redeem.redeeming') : t('redeem.redeemButton') }}
            </button>
          </form>

          <!-- Redeem Success -->
          <transition name="fade">
            <div
              v-if="redeemResult"
              class="mt-4 rounded-xl border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-800/50 dark:bg-emerald-900/20"
            >
              <div class="flex items-start gap-3">
                <Icon name="checkCircle" size="md" class="mt-0.5 text-emerald-600 dark:text-emerald-400" />
                <div class="text-sm text-emerald-700 dark:text-emerald-400">
                  <p class="font-medium">{{ t('redeem.redeemSuccess') }}</p>
                  <p class="mt-1">{{ redeemResult.message }}</p>
                </div>
              </div>
            </div>
          </transition>

          <!-- Redeem Error -->
          <transition name="fade">
            <div
              v-if="redeemError"
              class="mt-4 rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20"
            >
              <div class="flex items-start gap-3">
                <Icon name="exclamationCircle" size="md" class="mt-0.5 text-red-600 dark:text-red-400" />
                <p class="text-sm text-red-700 dark:text-red-400">{{ redeemError }}</p>
              </div>
            </div>
          </transition>
        </div>
      </div>

      <!-- Recent Activity (merged: redeem history + payment orders, last 30 days) -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('wallet.recentActivity') }}
          </h2>
          <p class="mt-0.5 text-xs text-gray-400 dark:text-dark-500">{{ t('wallet.last30Days') }}</p>
        </div>
        <div class="p-6">
          <!-- Loading State -->
          <div v-if="loadingHistory || loadingOrders" class="flex items-center justify-center py-8">
            <svg class="h-6 w-6 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          </div>

          <!-- Merged List -->
          <div v-else-if="mergedActivities.length > 0" class="space-y-3">
            <div
              v-for="item in mergedActivities"
              :key="item.key"
              class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800"
            >
              <!-- Payment Order Item -->
              <template v-if="item.kind === 'order'">
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-4">
                    <div
                      :class="[
                        'flex h-10 w-10 items-center justify-center rounded-xl',
                        isAlipayMethod(item.order.payment_method)
                          ? 'bg-blue-100 dark:bg-blue-900/30'
                          : isWechatMethod(item.order.payment_method)
                            ? 'bg-green-100 dark:bg-green-900/30'
                            : item.order.status === 'paid'
                              ? 'bg-emerald-100 dark:bg-emerald-900/30'
                              : item.order.status === 'pending'
                                ? 'bg-yellow-100 dark:bg-yellow-900/30'
                                : 'bg-gray-100 dark:bg-gray-800'
                      ]"
                    >
                      <!-- Alipay icon -->
                      <svg v-if="isAlipayMethod(item.order.payment_method)" class="h-5 w-5" viewBox="0 0 24 24">
                        <circle cx="12" cy="12" r="10" fill="#1677FF"/>
                        <path fill="white" d="M18.36 16.83c-1.05-.46-3.2-1.38-5.27-2.39.9-1.24 1.62-2.66 2.1-4.2H12V8.86h3.6V7.85H12V5.47a.41.41 0 00-.41-.41h-1.33v2.79H7v1.01h3.22v1.38H7.08v1h6.88c-.38 1.2-.93 2.3-1.62 3.2-1.68-.71-3.5-1.31-5.1-1.31-2.48 0-4.08 1.18-4.08 3.05 0 1.96 1.54 3.38 4.59 3.38 2.24 0 4.29-.97 5.87-2.52 1.81.93 4.62 2.2 5.2 2.46.2.1.4 0 .51-.18l.5-1.05c.1-.17 0-.4-.21-.5z"/>
                        <path fill="white" d="M8.5 17.25c-2.22 0-3.13-.88-3.13-1.98 0-1.06.87-1.92 2.6-1.92 1.42 0 3.03.57 4.6 1.32-1.3 1.61-2.69 2.58-4.07 2.58z"/>
                      </svg>
                      <!-- WeChat icon -->
                      <svg v-else-if="isWechatMethod(item.order.payment_method)" class="h-5 w-5 text-green-600 dark:text-green-400" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M9.5 4C5.36 4 2 6.69 2 10c0 1.89 1.08 3.56 2.78 4.66l-.7 2.1 2.6-1.37C7.6 15.78 8.53 16 9.5 16c.26 0 .52-.01.77-.04A5.46 5.46 0 0010 14.5c0-3.04 2.69-5.5 6-5.5.34 0 .67.03 1 .07C16.36 6.08 13.22 4 9.5 4zM6.75 7.25a1 1 0 110 2 1 1 0 010-2zm5.5 0a1 1 0 110 2 1 1 0 010-2zM16 10c-2.76 0-5 2.01-5 4.5S13.24 19 16 19c.72 0 1.4-.14 2.02-.38L20.5 20l-.5-1.88C21.23 17.12 22 15.88 22 14.5c0-2.49-2.69-4.5-6-4.5zm-2 3a.75.75 0 110 1.5.75.75 0 010-1.5zm4 0a.75.75 0 110 1.5.75.75 0 010-1.5z"/>
                      </svg>
                      <!-- Default credit card icon -->
                      <Icon
                        v-else
                        name="creditCard"
                        size="md"
                        :class="
                          item.order.status === 'paid'
                            ? 'text-emerald-600 dark:text-emerald-400'
                            : item.order.status === 'pending'
                              ? 'text-yellow-600 dark:text-yellow-400'
                              : 'text-gray-400 dark:text-gray-500'
                        "
                      />
                    </div>
                    <div>
                      <p class="text-sm font-medium text-gray-900 dark:text-white">
                        {{ t('wallet.topUpOrder') }}
                        {{ paymentConfig?.currency === 'CNY' ? '\u00a5' : '$' }}{{ item.order.amount.toFixed(2) }}
                      </p>
                      <p class="text-xs text-gray-500 dark:text-dark-400">
                        {{ formatDateTime(item.order.created_at) }}
                      </p>
                    </div>
                  </div>
                  <div class="text-right">
                    <span
                      :class="[
                        'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                        item.order.status === 'paid'
                          ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                          : item.order.status === 'pending'
                            ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
                            : 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400'
                      ]"
                    >
                      {{ t(`wallet.status${item.order.status.charAt(0).toUpperCase() + item.order.status.slice(1)}`) }}
                    </span>
                    <p
                      class="mt-1 cursor-pointer font-mono text-xs text-gray-400 hover:text-primary-500 dark:text-dark-500 dark:hover:text-primary-400"
                      :title="t('wallet.clickToShowOrderNo')"
                      @click="toggleOrderNo(item.order.id)"
                    >
                      {{ expandedOrderIds.has(item.order.id) ? item.order.order_no : item.order.order_no.slice(-8) }}
                    </p>
                  </div>
                </div>
              </template>

              <!-- Redeem History Item -->
              <template v-else>
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-4">
                    <div
                      :class="[
                        'flex h-10 w-10 items-center justify-center rounded-xl',
                        isBalanceType(item.history.type)
                          ? item.history.value >= 0
                            ? 'bg-emerald-100 dark:bg-emerald-900/30'
                            : 'bg-red-100 dark:bg-red-900/30'
                          : isSubscriptionType(item.history.type)
                            ? 'bg-purple-100 dark:bg-purple-900/30'
                            : item.history.value >= 0
                              ? 'bg-blue-100 dark:bg-blue-900/30'
                              : 'bg-orange-100 dark:bg-orange-900/30'
                      ]"
                    >
                      <Icon
                        v-if="isBalanceType(item.history.type)"
                        name="dollar"
                        size="md"
                        :class="item.history.value >= 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400'"
                      />
                      <Icon
                        v-else-if="isSubscriptionType(item.history.type)"
                        name="badge"
                        size="md"
                        class="text-purple-600 dark:text-purple-400"
                      />
                      <Icon
                        v-else
                        name="bolt"
                        size="md"
                        :class="item.history.value >= 0 ? 'text-blue-600 dark:text-blue-400' : 'text-orange-600 dark:text-orange-400'"
                      />
                    </div>
                    <div>
                      <p class="text-sm font-medium text-gray-900 dark:text-white">
                        {{ getHistoryItemTitle(item.history) }}
                      </p>
                      <p class="text-xs text-gray-500 dark:text-dark-400">
                        {{ formatDateTime(item.history.used_at) }}
                      </p>
                    </div>
                  </div>
                  <div class="text-right">
                    <p
                      :class="[
                        'text-sm font-semibold',
                        isBalanceType(item.history.type)
                          ? item.history.value >= 0
                            ? 'text-emerald-600 dark:text-emerald-400'
                            : 'text-red-600 dark:text-red-400'
                          : isSubscriptionType(item.history.type)
                            ? 'text-purple-600 dark:text-purple-400'
                            : item.history.value >= 0
                              ? 'text-blue-600 dark:text-blue-400'
                              : 'text-orange-600 dark:text-orange-400'
                      ]"
                    >
                      {{ formatHistoryValue(item.history) }}
                    </p>
                    <p
                      v-if="!isAdminAdjustment(item.history.type)"
                      class="font-mono text-xs text-gray-400 dark:text-dark-500"
                    >
                      {{ item.history.code.slice(0, 8) }}...
                    </p>
                    <p v-else class="text-xs text-gray-400 dark:text-dark-500">
                      {{ t('redeem.adminAdjustment') }}
                    </p>
                  </div>
                </div>
              </template>
            </div>
          </div>

          <!-- Empty State -->
          <div v-else class="empty-state py-8">
            <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-800">
              <Icon name="clock" size="xl" class="text-gray-400 dark:text-dark-500" />
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('wallet.historyEmpty') }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { redeemAPI, paymentAPI, type RedeemHistoryItem } from '@/api'
import type { PaymentConfig, CreatePaymentResponse } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'
import QRCode from 'qrcode'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const subscriptionStore = useSubscriptionStore()

const user = computed(() => authStore.user)

// Payment state
const paymentConfig = ref<PaymentConfig | null>(null)
const selectedAmount = ref<number>(0)
const customAmount = ref<number | undefined>(undefined)
const selectedMethod = ref<string>('')
const creatingOrder = ref(false)
const paymentResult = ref<CreatePaymentResponse | null>(null)
const showPaymentDialog = ref(false)
const paymentSuccess = ref(false)
const pollCountdown = ref(300)
const qrCodeDataUrl = ref<string>('')
const paymentPaidInDialog = ref(false)
const manualChecking = ref(false)
let currentOrderId: number | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null
let countdownTimer: ReturnType<typeof setInterval> | null = null

// Redeem state
const redeemCode = ref('')
const submitting = ref(false)
const redeemResult = ref<{ message: string; type: string; value: number } | null>(null)
const redeemError = ref('')

// History state
const history = ref<RedeemHistoryItem[]>([])
const loadingHistory = ref(false)

// Order history state
const orders = ref<any[]>([])
const loadingOrders = ref(false)
const expandedOrderIds = ref<Set<number>>(new Set())

type MergedActivity =
  | { kind: 'order'; key: string; date: number; order: any }
  | { kind: 'history'; key: string; date: number; history: RedeemHistoryItem }

const mergedActivities = computed<MergedActivity[]>(() => {
  const thirtyDaysAgo = Date.now() - 30 * 24 * 60 * 60 * 1000
  const items: MergedActivity[] = []

  for (const order of orders.value) {
    const d = new Date(order.created_at).getTime()
    if (d >= thirtyDaysAgo) {
      items.push({ kind: 'order', key: `order-${order.id}`, date: d, order })
    }
  }

  for (const h of history.value) {
    const d = new Date(h.used_at).getTime()
    if (d >= thirtyDaysAgo) {
      items.push({ kind: 'history', key: `history-${h.id}`, date: d, history: h })
    }
  }

  items.sort((a, b) => b.date - a.date)
  return items
})

function toggleOrderNo(orderId: number) {
  const s = new Set(expandedOrderIds.value)
  if (s.has(orderId)) {
    s.delete(orderId)
  } else {
    s.add(orderId)
  }
  expandedOrderIds.value = s
}

const effectiveAmount = computed(() => {
  if (customAmount.value && customAmount.value > 0) return customAmount.value
  if (selectedAmount.value > 0) return selectedAmount.value
  return 0
})

const availableMethods = computed(() => {
  if (!paymentConfig.value) return []
  const methods: { value: string; label: string }[] = []
  if (paymentConfig.value.methods.alipay) methods.push({ value: 'alipay', label: t('wallet.methodAlipay') })
  if (paymentConfig.value.methods.alipay_f2f) methods.push({ value: 'alipay_f2f', label: t('wallet.methodAlipayF2f') })
  if (paymentConfig.value.methods.wechat) methods.push({ value: 'wechat', label: t('wallet.methodWechat') })
  if (paymentConfig.value.methods.epay_alipay) methods.push({ value: 'epay_alipay', label: t('wallet.methodEpayAlipay') })
  if (paymentConfig.value.methods.epay_wechat) methods.push({ value: 'epay_wechat', label: t('wallet.methodEpayWechat') })
  return methods
})

const canSubmitPayment = computed(() => {
  return effectiveAmount.value > 0 && selectedMethod.value !== ''
})

function selectPresetAmount(amount: number) {
  selectedAmount.value = amount
  customAmount.value = undefined
}

function onCustomAmountInput() {
  selectedAmount.value = 0
}

async function handleCreateOrder() {
  if (!canSubmitPayment.value) return
  creatingOrder.value = true
  paymentSuccess.value = false

  try {
    const result = await paymentAPI.createOrder({
      amount: effectiveAmount.value,
      payment_method: selectedMethod.value
    })
    paymentResult.value = result

    // Alipay page pay returns an HTML form — open in new window
    if (result.form_html) {
      const w = window.open('', '_blank')
      if (w) {
        w.document.write(result.form_html)
        w.document.close()
      }
    }

    // Generate QR code image from payment URL
    if (result.qr_code_url) {
      try {
        qrCodeDataUrl.value = await QRCode.toDataURL(result.qr_code_url, {
          width: 256,
          margin: 2,
          color: { dark: '#000000', light: '#ffffff' }
        })
      } catch {
        qrCodeDataUrl.value = ''
      }
    } else {
      qrCodeDataUrl.value = ''
    }

    showPaymentDialog.value = true
    paymentPaidInDialog.value = false
    pollCountdown.value = 300
    currentOrderId = result.order_id
    startPolling(result.order_id)
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('wallet.createOrderFailed'))
  } finally {
    creatingOrder.value = false
  }
}

function startPolling(orderId: number) {
  stopPolling()
  pollTimer = setInterval(async () => {
    try {
      const order = await paymentAPI.getOrderStatus(orderId)
      console.log('[payment poll] orderId=', orderId, 'status=', order.status)
      if (order.status === 'paid') {
        stopPolling()
        paymentPaidInDialog.value = true
        paymentSuccess.value = true
        await authStore.refreshUser()
        await fetchHistory()
        await fetchOrders()
        appStore.showSuccess(t('wallet.paymentSuccessTitle'))
      } else if (order.status === 'expired' || order.status === 'failed') {
        stopPolling()
        showPaymentDialog.value = false
        appStore.showError(t('wallet.paymentExpired'))
      }
    } catch (e) {
      console.warn('[payment poll] error:', e)
    }
  }, 3000)

  countdownTimer = setInterval(() => {
    pollCountdown.value--
    if (pollCountdown.value <= 0) {
      stopPolling()
      showPaymentDialog.value = false
    }
  }, 1000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

function closePaymentDialog() {
  stopPolling()
  showPaymentDialog.value = false
  paymentPaidInDialog.value = false
  qrCodeDataUrl.value = ''
  currentOrderId = null
}

async function manualCheckStatus() {
  if (!currentOrderId) return
  manualChecking.value = true
  try {
    const order = await paymentAPI.getOrderStatus(currentOrderId)
    console.log('[payment manual check] orderId=', currentOrderId, 'status=', order.status)
    if (order.status === 'paid') {
      stopPolling()
      paymentPaidInDialog.value = true
      paymentSuccess.value = true
      await authStore.refreshUser()
      await fetchHistory()
      await fetchOrders()
      appStore.showSuccess(t('wallet.paymentSuccessTitle'))
    } else {
      appStore.showError(t('wallet.notPaidYet'))
    }
  } catch (e) {
    console.warn('[payment manual check] error:', e)
    appStore.showError(t('wallet.checkFailed'))
  } finally {
    manualChecking.value = false
  }
}

// Redeem functions
const isAlipayMethod = (method: string) => ['alipay', 'alipay_f2f', 'epay_alipay'].includes(method)
const isWechatMethod = (method: string) => ['wechat', 'epay_wechat'].includes(method)

const isBalanceType = (type: string) => type === 'balance' || type === 'admin_balance'
const isSubscriptionType = (type: string) => type === 'subscription'
const isAdminAdjustment = (type: string) => type === 'admin_balance' || type === 'admin_concurrency'

const getHistoryItemTitle = (item: RedeemHistoryItem) => {
  if (item.type === 'balance') return t('redeem.balanceAddedRedeem')
  if (item.type === 'admin_balance') return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
  if (item.type === 'concurrency') return t('redeem.concurrencyAddedRedeem')
  if (item.type === 'admin_concurrency') return item.value >= 0 ? t('redeem.concurrencyAddedAdmin') : t('redeem.concurrencyReducedAdmin')
  if (item.type === 'subscription') return t('redeem.subscriptionAssigned')
  return t('common.unknown')
}

const formatHistoryValue = (item: RedeemHistoryItem) => {
  if (isBalanceType(item.type)) {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}$${item.value.toFixed(2)}`
  } else if (isSubscriptionType(item.type)) {
    const days = item.validity_days || Math.round(item.value)
    const groupName = item.group?.name || ''
    return groupName ? `${days}${t('redeem.days')} - ${groupName}` : `${days}${t('redeem.days')}`
  } else {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}${item.value} ${t('redeem.requests')}`
  }
}

async function fetchHistory() {
  loadingHistory.value = true
  try {
    history.value = await redeemAPI.getHistory()
  } catch {
    console.error('Failed to fetch history')
  } finally {
    loadingHistory.value = false
  }
}

async function handleRedeem() {
  if (!redeemCode.value.trim()) {
    appStore.showError(t('redeem.pleaseEnterCode'))
    return
  }

  submitting.value = true
  redeemError.value = ''
  redeemResult.value = null

  try {
    const result = await redeemAPI.redeem(redeemCode.value.trim())
    redeemResult.value = result
    await authStore.refreshUser()

    if (result.type === 'subscription') {
      try {
        await subscriptionStore.fetchActiveSubscriptions(true)
      } catch {
        appStore.showWarning(t('redeem.subscriptionRefreshFailed'))
      }
    }

    redeemCode.value = ''
    await fetchHistory()
    appStore.showSuccess(t('redeem.codeRedeemSuccess'))
  } catch (error: any) {
    redeemError.value = error.response?.data?.detail || t('redeem.failedToRedeem')
    appStore.showError(t('redeem.redeemFailed'))
  } finally {
    submitting.value = false
  }
}

async function loadPaymentConfig() {
  try {
    paymentConfig.value = await paymentAPI.getPaymentConfig()
    // Auto-select first available method
    if (availableMethods.value.length > 0) {
      selectedMethod.value = availableMethods.value[0].value
    }
  } catch {
    console.error('Failed to load payment config')
  }
}

async function fetchOrders() {
  loadingOrders.value = true
  try {
    const result = await paymentAPI.getOrders({ page: 1, page_size: 20 })
    orders.value = result.orders || []
  } catch {
    console.error('Failed to fetch orders')
  } finally {
    loadingOrders.value = false
  }
}

onMounted(() => {
  fetchHistory()
  loadPaymentConfig()
  fetchOrders()
})

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.25s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}
</style>
