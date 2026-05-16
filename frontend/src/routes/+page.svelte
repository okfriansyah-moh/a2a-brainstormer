<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { agentRegistryStore } from '$lib/stores/agentRegistryStore';
	import AgentSelector from '$lib/components/AgentSelector.svelte';
	import { getAgents, createSession } from '$lib/services/api';
	import type { LLMConfig } from '$lib/types';

	let idea = '';
	let selectedAgentIds: string[] = [];
	let roleOverrides: Record<string, string> = {};
	let skillOverrides: Record<string, string[]> = {};
	let modelOverrides: Record<string, string> = {};
	let maxIterations = 5;
	let submitting = false;
	let error = '';

	onMount(async () => {
		agentRegistryStore.setLoading(true);
		try {
			const agents = await getAgents();
			agentRegistryStore.setAgents(agents);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load agents.';
		} finally {
			agentRegistryStore.setLoading(false);
		}
	});

	$: canSubmit =
		idea.trim().length > 0 &&
		selectedAgentIds.length >= 2 &&
		maxIterations >= 1 &&
		maxIterations <= 20 &&
		!submitting;

	async function handleSubmit() {
		if (!canSubmit) return;
		submitting = true;
		error = '';
		try {
			// Build llm_overrides from model overrides (keep provider + credential from agent defaults)
			const llmOverrides: Record<string, Partial<LLMConfig>> = {};
			for (const [agentId, model] of Object.entries(modelOverrides)) {
				if (model.trim()) {
					llmOverrides[agentId] = { model: model.trim() };
				}
			}

			// Build role_overrides (only include where it differs from default)
			const resolvedRoleOverrides: Record<string, string> = Object.keys(roleOverrides).length > 0
				? roleOverrides
				: undefined as unknown as Record<string, string>;

			// Build skill_overrides — only include entries that have explicit overrides
			const resolvedSkillOverrides: Record<string, string[]> = Object.keys(skillOverrides).length > 0
				? skillOverrides
				: undefined as unknown as Record<string, string[]>;

			const response = await createSession({
				idea: idea.trim(),
				agent_ids: selectedAgentIds,
				max_iterations: maxIterations,
				role_overrides: resolvedRoleOverrides,
				llm_overrides: Object.keys(llmOverrides).length > 0 ? llmOverrides : undefined,
				skill_overrides: resolvedSkillOverrides
			});
			await goto(`/session/${response.id}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create session.';
		} finally {
			submitting = false;
		}
	}
</script>

<div class="min-h-screen bg-gray-50">
	<header class="border-b border-gray-200 bg-white px-8 py-4">
		<div class="flex items-center justify-between">
			<div>
				<h1 class="text-lg font-bold text-gray-900">a2a-brainstorm</h1>
				<p class="text-xs text-gray-500">Deterministic multi-agent design engine</p>
			</div>
			<nav class="flex items-center gap-3 text-sm text-gray-600">
				<a href="/agents" class="hover:text-gray-900">Agents</a>
				<a href="/skills" class="hover:text-gray-900">Skills</a>
			</nav>
		</div>
	</header>

	<main class="mx-auto max-w-2xl px-4 py-10">
		<div class="rounded-xl border border-gray-200 bg-white p-8 shadow-sm">
			<h2 class="mb-1 text-xl font-semibold text-gray-900">New Brainstorm Session</h2>
			<p class="mb-6 text-sm text-gray-500">
				Describe your idea and select at least 2 agents to begin the iteration pipeline.
			</p>

			{#if error}
				<div class="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
					{error}
				</div>
			{/if}

			<!-- Idea input -->
			<div class="mb-5">
				<label for="idea" class="mb-1 block text-sm font-medium text-gray-700">
					Idea <span class="text-red-500">*</span>
				</label>
				<textarea
					id="idea"
					class="w-full resize-none rounded-md border border-gray-300 px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					rows={4}
					placeholder="Describe the idea you want to brainstorm..."
					bind:value={idea}
					maxlength={4000}
				></textarea>
				<p class="mt-1 text-right text-xs text-gray-400">{idea.length}/4000</p>
			</div>

			<!-- Max iterations -->
			<div class="mb-5">
				<label for="max-iter" class="mb-1 block text-sm font-medium text-gray-700">
					Max Iterations
				</label>
				<input
					id="max-iter"
					type="number"
					class="w-24 rounded-md border border-gray-300 px-3 py-2 text-sm text-gray-900 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					min="1"
					max="20"
					bind:value={maxIterations}
				/>
				<p class="mt-1 text-xs text-gray-400">Between 1 and 20.</p>
			</div>

			<!-- Agent selection -->
			<div class="mb-6">
				<p class="mb-2 text-sm font-medium text-gray-700">
					Agents <span class="text-red-500">*</span>
					<span class="ml-1 text-xs font-normal text-gray-500">(select at least 2)</span>
				</p>
				<AgentSelector
					agents={$agentRegistryStore.agents}
					loading={$agentRegistryStore.loading}
					bind:selectedAgentIds
					bind:roleOverrides
					bind:skillOverrides
					bind:modelOverrides
				/>
			</div>

			<!-- Submit -->
			<button
				type="button"
				class="w-full rounded-md bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
				disabled={!canSubmit}
				on:click={handleSubmit}
			>
				{#if submitting}
					Starting session...
				{:else}
					Start Brainstorm Session
				{/if}
			</button>
		</div>
	</main>
</div>
