<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { temperatureClient } from '$lib/grpc-client';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import * as Select from '$lib/components/ui/select';
	import { Badge } from '$lib/components/ui/badge';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Button } from '$lib/components/ui/button';
	import { RefreshCw, Thermometer, Server, Cpu, HardDrive, Activity } from '@lucide/svelte';
	import type { TemperatureReading } from '$lib/proto/jacuzzi/v1/temperature/v1/temperature_pb';

	let clients = $state<string[]>([]);
	let selectedClient = $state('');
	let currentReadings = $state<TemperatureReading[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let refreshInterval: ReturnType<typeof setInterval>;
	let autoRefresh = $state(true);

	// Fetch available clients
	async function fetchClients() {
		try {
			// Get distinct clients by fetching current temperatures for empty client ID
			// This is a workaround since we don't have a dedicated GetClients RPC
			const history = await temperatureClient.getTemperatureHistory({
				clientId: '',
				sensorId: '',
				limit: 1000
			});
			
			const uniqueClients = new Set<string>();
			history.readings.forEach(r => uniqueClients.add(r.clientId));
			clients = Array.from(uniqueClients).sort();
			
			if (clients.length > 0 && !selectedClient) {
				selectedClient = clients[0];
			}
		} catch (err) {
			console.error('Failed to fetch clients:', err);
			error = 'Failed to fetch clients';
		}
	}

	// Fetch current temperatures for selected client
	async function fetchCurrentTemperatures() {
		if (!selectedClient) return;
		
		loading = true;
		error = null;
		
		try {
			const response = await temperatureClient.getCurrentTemperatures({
				clientId: selectedClient
			});
			currentReadings = response.readings;
		} catch (err) {
			console.error('Failed to fetch temperatures:', err);
			error = 'Failed to fetch temperature data';
		} finally {
			loading = false;
		}
	}

	// Group readings by sensor type
	function groupBySensorType(readings: TemperatureReading[]) {
		const groups: Record<string, TemperatureReading[]> = {};
		readings.forEach(reading => {
			const type = reading.sensorType || 'Other';
			if (!groups[type]) groups[type] = [];
			groups[type].push(reading);
		});
		return groups;
	}

	// Get icon for sensor type
	function getSensorIcon(type: string) {
		switch (type.toUpperCase()) {
			case 'CPU': return Cpu;
			case 'GPU': return Activity;
			case 'DISK': return HardDrive;
			default: return Thermometer;
		}
	}

	// Get temperature color
	function getTemperatureColor(temp: number): string {
		if (temp >= 80) return 'text-red-600';
		if (temp >= 70) return 'text-orange-500';
		if (temp >= 60) return 'text-yellow-500';
		return 'text-green-600';
	}

	// Format timestamp
	function formatTimestamp(timestamp: any): string {
		if (!timestamp) return 'Unknown';
		const date = timestamp.toDate ? timestamp.toDate() : new Date(timestamp);
		return date.toLocaleString();
	}

	// Setup auto-refresh
	function setupAutoRefresh() {
		if (autoRefresh && selectedClient) {
			refreshInterval = setInterval(() => {
				fetchCurrentTemperatures();
			}, 5000); // Refresh every 5 seconds
		}
	}

	// Handle client selection
	$effect(() => {
		if (selectedClient) {
			fetchCurrentTemperatures();
		}
	});

	// Toggle auto-refresh
	function toggleAutoRefresh() {
		autoRefresh = !autoRefresh;
		if (!autoRefresh && refreshInterval) {
			clearInterval(refreshInterval);
		} else {
			setupAutoRefresh();
		}
	}

	onMount(async () => {
		await fetchClients();
		if (selectedClient) {
			await fetchCurrentTemperatures();
			setupAutoRefresh();
		}
	});

	onDestroy(() => {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}
	});

	$effect(() => {
		if (selectedClient && autoRefresh) {
			if (refreshInterval) clearInterval(refreshInterval);
			setupAutoRefresh();
		}
	});

	const groupedReadings = $derived(groupBySensorType(currentReadings));
</script>

<div class="max-w-7xl">
	<div class="mb-8">
		<h1 class="text-3xl font-bold mb-2">Temperature Dashboard</h1>
		<p class="text-muted-foreground">Real-time hardware temperature monitoring</p>
	</div>

	{#if error}
		<Alert class="mb-4" variant="destructive">
			<AlertDescription>{error}</AlertDescription>
		</Alert>
	{/if}

	<div class="flex gap-4 mb-6 items-center flex-wrap">
		<div class="flex items-center gap-2">
			<Server class="h-4 w-4 text-muted-foreground" />
			<Select.Root type="single" bind:value={selectedClient}>
				<Select.Trigger class="w-[200px]">
					{selectedClient || "Select a client"}
				</Select.Trigger>
				<Select.Content>
					{#each clients as client}
						<Select.Item value={client} label={client}>
							{client}
						</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>

		<div class="flex gap-2 ml-auto">
			<Button 
				variant="outline" 
				size="sm" 
				onclick={fetchCurrentTemperatures}
				disabled={!selectedClient || loading}
			>
				<RefreshCw class="h-4 w-4 mr-2" />
				Refresh
			</Button>
			<Button 
				variant={autoRefresh ? "default" : "outline"}
				size="sm" 
				onclick={toggleAutoRefresh}
			>
				{autoRefresh ? 'Auto-refresh ON' : 'Auto-refresh OFF'}
			</Button>
		</div>
	</div>

	{#if selectedClient}
		<Tabs value="current" class="space-y-4">
			<TabsList>
				<TabsTrigger value="current">Current Temperatures</TabsTrigger>
				<TabsTrigger value="history" disabled>History (Coming Soon)</TabsTrigger>
			</TabsList>

			<TabsContent value="current" class="space-y-4">
				{#if loading}
					<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
						{#each [1, 2, 3, 4, 5, 6] as i}
							<Card>
								<CardHeader>
									<Skeleton class="h-4 w-[100px]" />
									<Skeleton class="h-4 w-[150px]" />
								</CardHeader>
								<CardContent>
									<Skeleton class="h-8 w-[80px]" />
								</CardContent>
							</Card>
						{/each}
					</div>
				{:else if currentReadings.length === 0}
					<Card>
						<CardContent class="text-center py-8">
							<Thermometer class="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
							<p class="text-muted-foreground">No temperature data available for this client.</p>
						</CardContent>
					</Card>
				{:else}
					<div class="space-y-6">
						{#each Object.entries(groupedReadings) as [type, readings]}
							{@const Icon = getSensorIcon(type)}
							<div>
								<div class="flex items-center gap-2 mb-3">
									<Icon class="h-5 w-5" />
									<h3 class="text-lg font-semibold">{type} Sensors</h3>
									<Badge variant="secondary">{readings.length}</Badge>
								</div>
								
								<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
									{#each readings as reading}
										<Card>
											<CardHeader class="pb-3">
												<CardTitle class="text-base">{reading.sensorName}</CardTitle>
												<CardDescription class="text-xs">
													{reading.sensorId}
												</CardDescription>
											</CardHeader>
											<CardContent>
												<div class="flex items-baseline justify-between">
													<span class="text-3xl font-bold {getTemperatureColor(reading.temperatureCelsius)}">
														{reading.temperatureCelsius.toFixed(1)}°C
													</span>
													<span class="text-xs text-muted-foreground">
														{formatTimestamp(reading.timestamp)}
													</span>
												</div>
											</CardContent>
										</Card>
									{/each}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</TabsContent>
		</Tabs>
	{:else}
		<Card>
			<CardContent class="text-center py-8">
				<Server class="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
				<p class="text-muted-foreground">Select a client to view temperature data.</p>
			</CardContent>
		</Card>
	{/if}
</div>