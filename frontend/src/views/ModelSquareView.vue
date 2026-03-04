<template>
  <AppLayout>
    <div class="model-table-section px-2 md:px-6 py-6 max-w-[1400px] mx-auto">
      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-24">
        <LoadingSpinner />
      </div>

      <template v-else>
        <!-- Toolbar: Search + Filters + Count -->
        <div class="mb-4">
          <div class="flex flex-wrap items-center gap-2">
            <!-- Search Input -->
            <div class="relative flex-1 min-w-[200px]">
              <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 dark:text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8" />
                <path d="m21 21-4.3-4.3" stroke-linecap="round" />
              </svg>
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('modelSquare.searchPlaceholder')"
                class="w-full pl-9 pr-4 py-2 rounded-lg text-sm bg-white dark:bg-dark-800 border border-gray-200 dark:border-dark-600 text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500/30 transition-all"
              />
            </div>

            <!-- Provider Filter Dropdown -->
            <div class="relative" ref="providerDropdownRef">
              <button
                @click="showProviderDropdown = !showProviderDropdown"
                class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all bg-white dark:bg-dark-800 text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white border border-gray-200 dark:border-dark-600 hover:border-gray-300 dark:hover:border-dark-500"
                :class="{ '!border-primary-500 !text-primary-600 dark:!text-primary-400': selectedProvider }"
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3" />
                </svg>
                {{ selectedProvider || t('modelSquare.provider') }}
                <svg class="w-3.5 h-3.5 transition-transform" :class="{ 'rotate-180': showProviderDropdown }" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path d="m6 9 6 6 6-6" />
                </svg>
              </button>
              <transition name="dropdown">
                <div v-if="showProviderDropdown" class="absolute top-full left-0 mt-1 z-50 w-56 max-h-72 overflow-y-auto bg-white dark:bg-dark-800 rounded-xl border border-gray-200 dark:border-dark-700 shadow-lg py-1">
                  <button
                    @click="selectedProvider = ''; showProviderDropdown = false"
                    class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-dark-700 transition-colors"
                    :class="selectedProvider === '' ? 'text-primary-600 dark:text-primary-400 font-medium' : 'text-gray-700 dark:text-gray-300'"
                  >
                    {{ t('modelSquare.allProviders') }}
                  </button>
                  <button
                    v-for="provider in providerList"
                    :key="provider.name"
                    @click="selectedProvider = provider.name; showProviderDropdown = false"
                    class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-dark-700 transition-colors flex items-center justify-between"
                    :class="selectedProvider === provider.name ? 'text-primary-600 dark:text-primary-400 font-medium' : 'text-gray-700 dark:text-gray-300'"
                  >
                    <span class="flex items-center gap-2">
                      <span class="w-2 h-2 rounded-full flex-shrink-0" :style="{ backgroundColor: providerDotColor(provider.name) }"></span>
                      {{ provider.name }}
                    </span>
                    <span class="text-xs text-gray-400 dark:text-gray-500">{{ provider.count }}</span>
                  </button>
                </div>
              </transition>
            </div>

            <!-- Type Filter Dropdown -->
            <div class="relative" ref="modeDropdownRef">
              <button
                @click="showModeDropdown = !showModeDropdown"
                class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all bg-white dark:bg-dark-800 text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white border border-gray-200 dark:border-dark-600 hover:border-gray-300 dark:hover:border-dark-500"
                :class="{ '!border-primary-500 !text-primary-600 dark:!text-primary-400': selectedMode }"
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path d="M9.937 15.5A2 2 0 0 0 8.5 14.063l-6.135-1.582a.5.5 0 0 1 0-.962L8.5 9.936A2 2 0 0 0 9.937 8.5l1.582-6.135a.5.5 0 0 1 .963 0L14.063 8.5A2 2 0 0 0 15.5 9.937l6.135 1.581a.5.5 0 0 1 0 .964L15.5 14.063a2 2 0 0 0-1.437 1.437l-1.582 6.135a.5.5 0 0 1-.963 0z" />
                  <path d="M20 3v4" /><path d="M22 5h-4" />
                  <path d="M4 17v2" /><path d="M5 18H3" />
                </svg>
                {{ selectedMode ? modeBadgeLabel(selectedMode) : t('modelSquare.type') }}
                <svg class="w-3.5 h-3.5 transition-transform" :class="{ 'rotate-180': showModeDropdown }" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path d="m6 9 6 6 6-6" />
                </svg>
              </button>
              <transition name="dropdown">
                <div v-if="showModeDropdown" class="absolute top-full left-0 mt-1 z-50 w-48 bg-white dark:bg-dark-800 rounded-xl border border-gray-200 dark:border-dark-700 shadow-lg py-1">
                  <button
                    @click="selectedMode = ''; showModeDropdown = false"
                    class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-dark-700 transition-colors"
                    :class="selectedMode === '' ? 'text-primary-600 dark:text-primary-400 font-medium' : 'text-gray-700 dark:text-gray-300'"
                  >
                    {{ t('modelSquare.filterByType') }}
                  </button>
                  <button
                    v-for="mode in modeList"
                    :key="mode.name"
                    @click="selectedMode = mode.name; showModeDropdown = false"
                    class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-dark-700 transition-colors flex items-center justify-between"
                    :class="selectedMode === mode.name ? 'text-primary-600 dark:text-primary-400 font-medium' : 'text-gray-700 dark:text-gray-300'"
                  >
                    <span>{{ modeBadgeLabel(mode.name) }}</span>
                    <span class="text-xs text-gray-400 dark:text-gray-500">{{ mode.count }}</span>
                  </button>
                </div>
              </transition>
            </div>

            <!-- Sort Dropdown -->
            <div class="relative" ref="sortDropdownRef">
              <button
                @click="showSortDropdown = !showSortDropdown"
                class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all bg-white dark:bg-dark-800 text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white border border-gray-200 dark:border-dark-600 hover:border-gray-300 dark:hover:border-dark-500"
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path d="m21 16-4 4-4-4" /><path d="M17 20V4" />
                  <path d="m3 8 4-4 4 4" /><path d="M7 4v16" />
                </svg>
                {{ sortLabels[sortKey] || t('modelSquare.modelName') }}
                <svg class="w-3.5 h-3.5 transition-transform" :class="{ 'rotate-180': showSortDropdown }" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path d="m6 9 6 6 6-6" />
                </svg>
              </button>
              <transition name="dropdown">
                <div v-if="showSortDropdown" class="absolute top-full left-0 mt-1 z-50 w-48 bg-white dark:bg-dark-800 rounded-xl border border-gray-200 dark:border-dark-700 shadow-lg py-1">
                  <button
                    v-for="(label, key) in sortLabels"
                    :key="key"
                    @click="toggleSort(key); showSortDropdown = false"
                    class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-dark-700 transition-colors flex items-center justify-between"
                    :class="sortKey === key ? 'text-primary-600 dark:text-primary-400 font-medium' : 'text-gray-700 dark:text-gray-300'"
                  >
                    <span>{{ label }}</span>
                    <svg v-if="sortKey === key" class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path v-if="sortDir === 'asc'" d="m3 8 4-4 4 4" /><path v-if="sortDir === 'asc'" d="M7 4v16" />
                      <path v-if="sortDir === 'desc'" d="m21 16-4 4-4-4" /><path v-if="sortDir === 'desc'" d="M17 20V4" />
                    </svg>
                  </button>
                </div>
              </transition>
            </div>

            <!-- Model Count -->
            <div class="ml-auto text-sm text-gray-500 dark:text-gray-400 font-mono">
              {{ filteredModels.length.toLocaleString() }} {{ t('modelSquare.modelsAvailable') }}
            </div>
          </div>
        </div>

        <!-- Table -->
        <div v-if="filteredModels.length > 0" class="rounded-xl border border-gray-200 dark:border-dark-700 overflow-hidden bg-white dark:bg-dark-800/50">
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-gray-50/80 dark:bg-dark-800/80">
                  <th
                    @click="toggleSort('provider')"
                    class="cursor-pointer hover:bg-gray-100 dark:hover:bg-dark-700 transition-colors px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 whitespace-nowrap"
                  >
                    <div class="flex items-center gap-1.5">
                      {{ t('modelSquare.provider') }}
                      <svg class="w-3 h-3" :class="sortKey === 'provider' ? 'opacity-100 text-primary-500' : 'opacity-30'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path d="m21 16-4 4-4-4" /><path d="M17 20V4" />
                        <path d="m3 8 4-4 4 4" /><path d="M7 4v16" />
                      </svg>
                    </div>
                  </th>
                  <th
                    @click="toggleSort('id')"
                    class="cursor-pointer hover:bg-gray-100 dark:hover:bg-dark-700 transition-colors px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 whitespace-nowrap"
                  >
                    <div class="flex items-center gap-1.5">
                      Model ID
                      <svg class="w-3 h-3" :class="sortKey === 'id' ? 'opacity-100 text-primary-500' : 'opacity-30'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path d="m21 16-4 4-4-4" /><path d="M17 20V4" />
                        <path d="m3 8 4-4 4 4" /><path d="M7 4v16" />
                      </svg>
                    </div>
                  </th>
                  <th
                    @click="toggleSort('input_price')"
                    class="cursor-pointer hover:bg-gray-100 dark:hover:bg-dark-700 transition-colors px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 whitespace-nowrap"
                  >
                    <div class="flex items-center gap-1.5">
                      {{ t('modelSquare.input') }} $/M
                      <svg class="w-3 h-3" :class="sortKey === 'input_price' ? 'opacity-100 text-primary-500' : 'opacity-30'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path d="m21 16-4 4-4-4" /><path d="M17 20V4" />
                        <path d="m3 8 4-4 4 4" /><path d="M7 4v16" />
                      </svg>
                    </div>
                  </th>
                  <th
                    @click="toggleSort('output_price')"
                    class="cursor-pointer hover:bg-gray-100 dark:hover:bg-dark-700 transition-colors px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 whitespace-nowrap"
                  >
                    <div class="flex items-center gap-1.5">
                      {{ t('modelSquare.output') }} $/M
                      <svg class="w-3 h-3" :class="sortKey === 'output_price' ? 'opacity-100 text-primary-500' : 'opacity-30'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path d="m21 16-4 4-4-4" /><path d="M17 20V4" />
                        <path d="m3 8 4-4 4 4" /><path d="M7 4v16" />
                      </svg>
                    </div>
                  </th>
                  <th
                    @click="toggleSort('cache_read_price')"
                    class="cursor-pointer hover:bg-gray-100 dark:hover:bg-dark-700 transition-colors px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 whitespace-nowrap"
                  >
                    <div class="flex items-center gap-1.5">
                      {{ t('modelSquare.cacheRead') }} $/M
                      <svg class="w-3 h-3" :class="sortKey === 'cache_read_price' ? 'opacity-100 text-primary-500' : 'opacity-30'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path d="m21 16-4 4-4-4" /><path d="M17 20V4" />
                        <path d="m3 8 4-4 4 4" /><path d="M7 4v16" />
                      </svg>
                    </div>
                  </th>
                  <th
                    @click="toggleSort('cache_create_price')"
                    class="cursor-pointer hover:bg-gray-100 dark:hover:bg-dark-700 transition-colors px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 whitespace-nowrap"
                  >
                    <div class="flex items-center gap-1.5">
                      {{ t('modelSquare.cacheCreate') }} $/M
                      <svg class="w-3 h-3" :class="sortKey === 'cache_create_price' ? 'opacity-100 text-primary-500' : 'opacity-30'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path d="m21 16-4 4-4-4" /><path d="M17 20V4" />
                        <path d="m3 8 4-4 4 4" /><path d="M7 4v16" />
                      </svg>
                    </div>
                  </th>
                  <th class="px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 whitespace-nowrap">
                    {{ t('modelSquare.type') }}
                  </th>
                  <th class="w-10 px-2 py-3"></th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="model in paginatedModels"
                  :key="model.id"
                  class="group cursor-pointer border-t border-gray-100 dark:border-dark-700/50 hover:bg-gray-50 dark:hover:bg-dark-700/30 transition-colors"
                  @click="copyModelName(model.id)"
                >
                  <!-- Provider -->
                  <td class="px-4 py-2.5 whitespace-nowrap">
                    <div class="flex items-center gap-2">
                      <span class="w-2 h-2 rounded-full flex-shrink-0" :style="{ backgroundColor: providerDotColor(model.provider) }"></span>
                      <span class="text-gray-600 dark:text-gray-300 text-sm">{{ model.provider }}</span>
                    </div>
                  </td>
                  <!-- Model ID -->
                  <td class="px-4 py-2.5">
                    <div class="flex items-center gap-2">
                      <span class="font-medium text-gray-900 dark:text-white text-sm">{{ model.id }}</span>
                      <transition name="fade">
                        <span
                          v-if="copiedModel === model.id"
                          class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                        >
                          {{ t('modelSquare.copied') }}
                        </span>
                      </transition>
                    </div>
                  </td>
                  <!-- Input Price -->
                  <td class="px-4 py-2.5 font-mono whitespace-nowrap">
                    <span class="text-gray-900 dark:text-gray-100">{{ formatPrice(model.input_price) }}</span>
                  </td>
                  <!-- Output Price -->
                  <td class="px-4 py-2.5 font-mono whitespace-nowrap">
                    <span class="text-gray-900 dark:text-gray-100">{{ formatPrice(model.output_price) }}</span>
                  </td>
                  <!-- Cache Read -->
                  <td class="px-4 py-2.5 font-mono whitespace-nowrap">
                    <span v-if="model.cache_read_price != null" class="text-gray-900 dark:text-gray-100">{{ formatPrice(model.cache_read_price) }}</span>
                    <span v-else class="text-gray-300 dark:text-gray-600">&mdash;</span>
                  </td>
                  <!-- Cache Create -->
                  <td class="px-4 py-2.5 font-mono whitespace-nowrap">
                    <span v-if="model.cache_create_price != null" class="text-gray-900 dark:text-gray-100">{{ formatPrice(model.cache_create_price) }}</span>
                    <span v-else class="text-gray-300 dark:text-gray-600">&mdash;</span>
                  </td>
                  <!-- Type -->
                  <td class="px-4 py-2.5 whitespace-nowrap">
                    <span class="inline-flex items-center rounded-md px-2 py-0.5 text-[11px] font-medium" :class="modeBadgeClass(model.mode)">
                      {{ modeBadgeLabel(model.mode) }}
                    </span>
                  </td>
                  <!-- Copy Button -->
                  <td class="px-2 py-2.5 text-center">
                    <button
                      class="p-1.5 rounded-md hover:bg-gray-200 dark:hover:bg-dark-600 text-gray-400 dark:text-gray-500 hover:text-gray-700 dark:hover:text-gray-200 transition-all opacity-0 group-hover:opacity-100"
                      :title="t('modelSquare.copied')"
                      @click.stop="copyModelName(model.id)"
                    >
                      <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
                        <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
                      </svg>
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Pagination -->
          <div class="mt-0 flex items-center justify-between gap-4 px-4 py-3 border-t border-gray-100 dark:border-dark-700">
            <div class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('modelSquare.showing', { count: Math.min(currentPage * pageSize, filteredModels.length), total: filteredModels.length }) }}
              <span class="text-xs text-gray-400 dark:text-gray-500 ml-2">{{ t('modelSquare.priceUnit') }}</span>
            </div>
            <div v-if="totalPages > 1" class="flex items-center gap-1">
              <!-- First Page -->
              <button
                @click="currentPage = 1"
                :disabled="currentPage === 1"
                class="p-2 rounded-md border border-gray-200 dark:border-dark-600 bg-white dark:bg-dark-800 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-dark-700 hover:text-gray-900 dark:hover:text-white disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-white dark:disabled:hover:bg-dark-800 disabled:hover:text-gray-600 dark:disabled:hover:text-gray-300 transition-all"
                :title="t('modelSquare.prev')"
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path d="m11 17-5-5 5-5" /><path d="m18 17-5-5 5-5" />
                </svg>
              </button>
              <!-- Prev Page -->
              <button
                @click="currentPage = Math.max(1, currentPage - 1)"
                :disabled="currentPage === 1"
                class="p-2 rounded-md border border-gray-200 dark:border-dark-600 bg-white dark:bg-dark-800 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-dark-700 hover:text-gray-900 dark:hover:text-white disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-white dark:disabled:hover:bg-dark-800 disabled:hover:text-gray-600 dark:disabled:hover:text-gray-300 transition-all"
                :title="t('modelSquare.prev')"
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path d="m15 18-6-6 6-6" />
                </svg>
              </button>
              <!-- Page Numbers -->
              <div class="flex items-center gap-1 mx-2">
                <template v-for="page in visiblePages" :key="page">
                  <button
                    v-if="page !== '...'"
                    @click="currentPage = page as number"
                    class="min-w-[36px] h-9 px-3 rounded-md text-sm font-medium transition-all"
                    :class="currentPage === page
                      ? 'bg-primary-500 text-white shadow-sm'
                      : 'border border-gray-200 dark:border-dark-600 bg-white dark:bg-dark-800 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-dark-700 hover:text-gray-900 dark:hover:text-white'"
                  >
                    {{ page }}
                  </button>
                  <span v-else class="px-2 text-gray-400 dark:text-gray-500">...</span>
                </template>
              </div>
              <!-- Next Page -->
              <button
                @click="currentPage = Math.min(totalPages, currentPage + 1)"
                :disabled="currentPage === totalPages"
                class="p-2 rounded-md border border-gray-200 dark:border-dark-600 bg-white dark:bg-dark-800 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-dark-700 hover:text-gray-900 dark:hover:text-white disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-white dark:disabled:hover:bg-dark-800 disabled:hover:text-gray-600 dark:disabled:hover:text-gray-300 transition-all"
                :title="t('modelSquare.next')"
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path d="m9 18 6-6-6-6" />
                </svg>
              </button>
              <!-- Last Page -->
              <button
                @click="currentPage = totalPages"
                :disabled="currentPage === totalPages"
                class="p-2 rounded-md border border-gray-200 dark:border-dark-600 bg-white dark:bg-dark-800 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-dark-700 hover:text-gray-900 dark:hover:text-white disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-white dark:disabled:hover:bg-dark-800 disabled:hover:text-gray-600 dark:disabled:hover:text-gray-300 transition-all"
                :title="t('modelSquare.next')"
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path d="m6 17 5-5-5-5" /><path d="m13 17 5-5-5-5" />
                </svg>
              </button>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-else class="rounded-xl border border-gray-200 dark:border-dark-700 bg-white dark:bg-dark-800/50 py-16 text-center">
          <svg class="mx-auto h-12 w-12 text-gray-300 dark:text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <p class="mt-4 text-sm text-gray-500 dark:text-gray-400">{{ t('modelSquare.noModels') }}</p>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { getModelPricing, type ModelPricingDisplay } from '@/api/models'

const { t } = useI18n()

const loading = ref(true)
const models = ref<ModelPricingDisplay[]>([])
const updatedAt = ref('')
const searchQuery = ref('')
const selectedProvider = ref('')
const selectedMode = ref('')
const copiedModel = ref('')
const currentPage = ref(1)
const pageSize = 100

// Dropdowns
const showProviderDropdown = ref(false)
const showModeDropdown = ref(false)
const showSortDropdown = ref(false)
const providerDropdownRef = ref<HTMLElement>()
const modeDropdownRef = ref<HTMLElement>()
const sortDropdownRef = ref<HTMLElement>()

// Sorting
const sortKey = ref<string>('')
const sortDir = ref<'asc' | 'desc'>('asc')

const sortLabels: Record<string, string> = {
  provider: 'Provider',
  id: 'Model ID',
  input_price: 'Input $/M',
  output_price: 'Output $/M',
  cache_read_price: 'Cache Read $/M',
  cache_create_price: 'Cache Write $/M',
}

let copyTimer: ReturnType<typeof setTimeout> | null = null

// Close dropdowns when clicking outside
function handleClickOutside(e: MouseEvent) {
  const target = e.target as Node
  if (providerDropdownRef.value && !providerDropdownRef.value.contains(target)) {
    showProviderDropdown.value = false
  }
  if (modeDropdownRef.value && !modeDropdownRef.value.contains(target)) {
    showModeDropdown.value = false
  }
  if (sortDropdownRef.value && !sortDropdownRef.value.contains(target)) {
    showSortDropdown.value = false
  }
}

onMounted(async () => {
  document.addEventListener('click', handleClickOutside)
  try {
    const res = await getModelPricing()
    models.value = res.models || []
    updatedAt.value = res.updated_at || ''
  } catch (e) {
    console.error('Failed to fetch model pricing:', e)
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

const providerList = computed(() => {
  const map = new Map<string, number>()
  for (const m of models.value) {
    map.set(m.provider, (map.get(m.provider) || 0) + 1)
  }
  return Array.from(map.entries())
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count)
})

const modeList = computed(() => {
  const map = new Map<string, number>()
  for (const m of models.value) {
    const mode = m.mode || 'chat'
    map.set(mode, (map.get(mode) || 0) + 1)
  }
  return Array.from(map.entries())
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count)
})

const filteredModels = computed(() => {
  let list = models.value
  if (selectedProvider.value) {
    list = list.filter((m) => m.provider === selectedProvider.value)
  }
  if (selectedMode.value) {
    list = list.filter((m) => (m.mode || 'chat') === selectedMode.value)
  }
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase()
    list = list.filter((m) => m.id.toLowerCase().includes(q) || m.provider.toLowerCase().includes(q))
  }
  // Sorting
  if (sortKey.value) {
    const key = sortKey.value
    const dir = sortDir.value === 'asc' ? 1 : -1
    list = [...list].sort((a, b) => {
      const av = (a as any)[key]
      const bv = (b as any)[key]
      if (av == null && bv == null) return 0
      if (av == null) return 1
      if (bv == null) return -1
      if (typeof av === 'string') return av.localeCompare(bv) * dir
      return (av - bv) * dir
    })
  }
  return list
})

const totalPages = computed(() => Math.max(1, Math.ceil(filteredModels.value.length / pageSize)))

const paginatedModels = computed(() => {
  const start = (currentPage.value - 1) * pageSize
  return filteredModels.value.slice(start, start + pageSize)
})

// Generate visible page numbers with ellipsis
const visiblePages = computed(() => {
  const total = totalPages.value
  const current = currentPage.value
  const pages: (number | string)[] = []

  if (total <= 7) {
    for (let i = 1; i <= total; i++) pages.push(i)
  } else {
    pages.push(1)
    if (current > 3) pages.push('...')
    const start = Math.max(2, current - 1)
    const end = Math.min(total - 1, current + 1)
    for (let i = start; i <= end; i++) pages.push(i)
    if (current < total - 2) pages.push('...')
    pages.push(total)
  }

  return pages
})

// Reset page when filters change
watch([searchQuery, selectedProvider, selectedMode, sortKey, sortDir], () => {
  currentPage.value = 1
})

function toggleSort(key: string) {
  if (sortKey.value === key) {
    if (sortDir.value === 'asc') {
      sortDir.value = 'desc'
    } else {
      sortKey.value = ''
      sortDir.value = 'asc'
    }
  } else {
    sortKey.value = key
    sortDir.value = 'asc'
  }
}

function formatPrice(price: number): string {
  if (price === 0) return '$0'
  if (price < 0.001) return `$${price.toFixed(5)}`
  if (price < 0.01) return `$${price.toFixed(4)}`
  if (price < 1) return `$${price.toFixed(3)}`
  return `$${price.toFixed(2)}`
}

function modeBadgeLabel(mode: string): string {
  const labels: Record<string, string> = {
    chat: 'Chat',
    completion: 'Completion',
    embedding: 'Embedding',
    image_generation: 'Image',
    audio_transcription: 'Audio',
    audio_speech: 'TTS',
    moderation: 'Moderation',
    rerank: 'Rerank',
  }
  return labels[mode] || mode
}

function modeBadgeClass(mode: string): string {
  const classes: Record<string, string> = {
    chat: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
    completion: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
    embedding: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
    image_generation: 'bg-pink-100 text-pink-700 dark:bg-pink-900/30 dark:text-pink-400',
    audio_transcription: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
    audio_speech: 'bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-400',
    moderation: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
    rerank: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400',
  }
  return classes[mode] || 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-400'
}

const providerDotColors: Record<string, string> = {
  OpenAI: '#10B981',
  Anthropic: '#D4A574',
  Google: '#4285F4',
  DeepSeek: '#6366F1',
  Mistral: '#F59E0B',
  Meta: '#0EA5E9',
  Cohere: '#A855F7',
  Qwen: '#14B8A6',
  xAI: '#EF4444',
  Groq: '#F97316',
}

function providerDotColor(provider: string): string {
  return providerDotColors[provider] || '#9CA3AF'
}

async function copyModelName(modelId: string) {
  try {
    await navigator.clipboard.writeText(modelId)
  } catch {
    const ta = document.createElement('textarea')
    ta.value = modelId
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
  }
  copiedModel.value = modelId
  if (copyTimer) clearTimeout(copyTimer)
  copyTimer = setTimeout(() => { copiedModel.value = '' }, 2000)
}
</script>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

.dropdown-enter-active { transition: opacity 0.15s ease, transform 0.15s ease; }
.dropdown-leave-active { transition: opacity 0.1s ease, transform 0.1s ease; }
.dropdown-enter-from, .dropdown-leave-to { opacity: 0; transform: translateY(-4px) scale(0.98); }
</style>
