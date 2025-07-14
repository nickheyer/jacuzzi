<script lang="ts">
	import { onMount } from 'svelte';
	import { temperatureClient, clientClient } from '$lib/grpc-client';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import * as Select from '$lib/components/ui/select';
	import { Button } from '$lib/components/ui/button';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Label } from '$lib/components/ui/label';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import { RefreshCw, TrendingUp, TrendingDown, Activity, Calendar } from '@lucide/svelte';
	import { ChartContainer } from '$lib/components/ui/chart';
	import type { TemperatureReading, TemperatureStats } from '$lib/proto/jacuzzi/v1/temperature/v1/temperature_pb';
	import type { Client } from '$lib/proto/jacuzzi/v1/client/v1/client_pb';
	
	let clients = $state<Client[]>([]);
	let selectedClient = $state('');
	let selectedTimeRange = $state('1h');
	let loading = $state(false);
	let error = $state<string | null>(null);
	let temperatureHistory = $state<TemperatureReading[]>([]);
	let temperatureStats = $state<Record<string, TemperatureStats>>({});
	
	const timeRanges = [
		{ value: '1h', label: 'Last Hour' },
		{ value: '6h', label: 'Last 6 Hours' },
		{ value: '24h', label: 'Last 24 Hours' },
		{ value: '7d', label: 'Last 7 Days' },
		{ value: '30d', label: 'Last 30 Days' }
	];
	
	async function fetchClients() {
		try {
			const response = await clientClient.listClients({
				onlineOnly: false,
				limit: 100,
				offset: 0
			});
			clients = response.clients;
			
			if (clients.length > 0 && !selectedClient) {
				selectedClient = clients[0].id;
			}
		} catch (err) {
			console.error('Failed to fetch clients:', err);
			error = 'Failed to fetch clients';
		}
	}
	
	async function fetchData() {
		if (!selectedClient) return;
		
		loading = true;
		error = null;
		
		try {
			const now = new Date();
			const startTime = new Date();
			
			// Calculate start time based on selected range
			switch (selectedTimeRange) {
				case '1h':
					startTime.setHours(now.getHours() - 1);
					break;
				case '6h':
					startTime.setHours(now.getHours() - 6);
					break;
				case '24h':
					startTime.setDate(now.getDate() - 1);
					break;
				case '7d':
					startTime.setDate(now.getDate() - 7);
					break;
				case '30d':
					startTime.setDate(now.getDate() - 30);
					break;
			}
			
			// Fetch temperature history
			const historyResponse = await temperatureClient.getTemperatureHistory({
				clientId: selectedClient,
				sensorId: '',
				startTime: { seconds: BigInt(Math.floor(startTime.getTime() / 1000)), nanos: 0 },
				endTime: { seconds: BigInt(Math.floor(now.getTime() / 1000)), nanos: 0 },
				limit: 1000
			});
			temperatureHistory = historyResponse.readings;
			
			// Fetch temperature statistics
			const statsResponse = await temperatureClient.getTemperatureStats({
				clientId: selectedClient,
				sensorId: '',
				startTime: { seconds: BigInt(Math.floor(startTime.getTime() / 1000)), nanos: 0 },
				endTime: { seconds: BigInt(Math.floor(now.getTime() / 1000)), nanos: 0 }
			});
			temperatureStats = statsResponse.sensorStats;
		} catch (err) {
			console.error('Failed to fetch data:', err);
			error = 'Failed to fetch temperature data';
		} finally {
			loading = false;
		}
	}
	
	// Process data for charts
	function processChartData() {
		const sensorData: Record<string, { time: Date; value: number }[]> = {};
		
		temperatureHistory.forEach(reading => {
			const sensorKey = `${reading.sensorType}-${reading.sensorName}`;
			if (!sensorData[sensorKey]) {
				sensorData[sensorKey] = [];
			}
			
			const timestamp = reading.timestamp;
			const date = timestamp ? new Date(Number(timestamp.seconds) * 1000) : new Date();
			
			sensorData[sensorKey].push({
				time: date,
				value: reading.temperatureCelsius
			});
		});
		
		// Sort by time
		Object.keys(sensorData).forEach(key => {
			sensorData[key].sort((a, b) => a.time.getTime() - b.time.getTime());
		});
		
		return sensorData;
	}
	
	// Get sensor color
	function getSensorColor(index: number): string {
		const colors = [
			'hsl(var(--chart-1))',
			'hsl(var(--chart-2))',
			'hsl(var(--chart-3))',
			'hsl(var(--chart-4))',
			'hsl(var(--chart-5))'
		];
		return colors[index % colors.length];
	}
	
	function formatTimestamp(timestamp: any): string {
		if (!timestamp) return 'Unknown';
		const date = timestamp.seconds ? new Date(Number(timestamp.seconds) * 1000) : new Date();
		return date.toLocaleString();
	}
	
	onMount(() => {
		fetchClients();
	});
	
	$effect(() => {
		if (selectedClient && selectedTimeRange) {
			fetchData();
		}
	});
	
	const chartData = $derived(processChartData());
	const sensorKeys = $derived(Object.keys(chartData));
</script>

<div class="max-w-7xl">
	<div class="mb-8">
		<h1 class="text-3xl font-bold mb-2">Temperature Statistics</h1>
		<p class="text-muted-foreground">Analyze temperature trends and patterns</p>
	</div>
	
	{#if error}
		<Alert class="mb-4" variant="destructive">
			<AlertDescription>{error}</AlertDescription>
		</Alert>
	{/if}
	
	<div class="flex gap-4 mb-6 items-center">
		<div class="flex items-center gap-2">
			<Label>Client:</Label>
			<Select.Root type="single" bind:value={selectedClient}>
				<Select.Trigger class="w-[200px]">
					{clients.find(c => c.id === selectedClient)?.hostname || "Select a client"}
				</Select.Trigger>
				<Select.Content>
					{#each clients as client}
						<Select.Item value={client.id} label={client.hostname}>
							{client.hostname}
						</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>
		
		<div class="flex items-center gap-2">
			<Label>Time Range:</Label>
			<Select.Root type="single" bind:value={selectedTimeRange}>
				<Select.Trigger class="w-[150px]">
					{timeRanges.find(r => r.value === selectedTimeRange)?.label || "Select time range"}
				</Select.Trigger>
				<Select.Content>
					{#each timeRanges as range}
						<Select.Item value={range.value} label={range.label}>
							{range.label}
						</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>
		
		<Button 
			variant="outline" 
			size="sm"
			onclick={fetchData}
			disabled={!selectedClient || loading}
			class="ml-auto"
		>
			<RefreshCw class="h-4 w-4 mr-2" />
			Refresh
		</Button>
	</div>
	
	{#if loading}
		<div class="space-y-4">
			<Card>
				<CardHeader>
					<Skeleton class="h-6 w-[200px]" />
					<Skeleton class="h-4 w-[300px]" />
				</CardHeader>
				<CardContent>
					<Skeleton class="h-[300px] w-full" />
				</CardContent>
			</Card>
		</div>
	{:else if temperatureHistory.length === 0}
		<Card>
			<CardContent class="text-center py-8">
				<Activity class="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
				<p class="text-muted-foreground">No temperature data available for the selected time range.</p>
			</CardContent>
		</Card>
	{:else}
		<Tabs value="overview" class="space-y-4">
			<TabsList>
				<TabsTrigger value="overview">Overview</TabsTrigger>
				<TabsTrigger value="trends">Temperature Trends</TabsTrigger>
				<TabsTrigger value="heatmap">Heat Map</TabsTrigger>
			</TabsList>
			
			<TabsContent value="overview">
				<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
					{#each Object.entries(temperatureStats).slice(0, 4) as [sensorId, stats]}
						<Card>
							<CardHeader class="pb-2">
								<CardTitle class="text-sm font-medium">
									{temperatureHistory.find(r => r.sensorId === sensorId)?.sensorName || sensorId}
								</CardTitle>
							</CardHeader>
							<CardContent>
								<div class="space-y-2">
									<div class="flex items-center justify-between">
										<span class="text-xs text-muted-foreground">Average</span>
										<span class="text-sm font-bold">{stats.avgTemperature.toFixed(1)}°C</span>
									</div>
									<div class="flex items-center justify-between">
										<span class="text-xs text-muted-foreground">Min</span>
										<span class="text-sm">{stats.minTemperature.toFixed(1)}°C</span>
									</div>
									<div class="flex items-center justify-between">
										<span class="text-xs text-muted-foreground">Max</span>
										<span class="text-sm">{stats.maxTemperature.toFixed(1)}°C</span>
									</div>
									<div class="flex items-center justify-between">
										<span class="text-xs text-muted-foreground">Readings</span>
										<span class="text-sm">{stats.readingCount}</span>
									</div>
								</div>
							</CardContent>
						</Card>
					{/each}
				</div>
				
				<Card>
					<CardHeader>
						<CardTitle>Temperature Summary</CardTitle>
						<CardDescription>Statistics for all sensors in the selected time range</CardDescription>
					</CardHeader>
					<CardContent>
						<div class="space-y-3">
							{#each Object.entries(temperatureStats) as [sensorId, stats]}
								{@const sensor = temperatureHistory.find(r => r.sensorId === sensorId)}
								<div class="flex items-center justify-between p-3 bg-muted/50 rounded-lg">
									<div>
										<p class="font-medium">{sensor?.sensorName || sensorId}</p>
										<p class="text-sm text-muted-foreground">{sensor?.sensorType}</p>
									</div>
									<div class="flex gap-6 text-sm">
										<div class="text-center">
											<p class="text-muted-foreground">Min</p>
											<p class="font-mono">{stats.minTemperature.toFixed(1)}°</p>
										</div>
										<div class="text-center">
											<p class="text-muted-foreground">Avg</p>
											<p class="font-mono font-bold">{stats.avgTemperature.toFixed(1)}°</p>
										</div>
										<div class="text-center">
											<p class="text-muted-foreground">Max</p>
											<p class="font-mono">{stats.maxTemperature.toFixed(1)}°</p>
										</div>
									</div>
								</div>
							{/each}
						</div>
					</CardContent>
				</Card>
			</TabsContent>
			
			<TabsContent value="trends">
				<Card>
					<CardHeader>
						<CardTitle>Temperature Trends</CardTitle>
						<CardDescription>Temperature changes over time for each sensor</CardDescription>
					</CardHeader>
					<CardContent>
						<div class="h-[400px]">
							<svg viewBox="0 0 800 400" class="w-full h-full">
								<!-- Grid lines -->
								{#each Array(5) as _, i}
									<line 
										x1="50" 
										y1={50 + i * 75} 
										x2="750" 
										y2={50 + i * 75} 
										stroke="hsl(var(--border))" 
										stroke-dasharray="5,5"
									/>
								{/each}
								
								<!-- Temperature lines -->
								{#each sensorKeys as key, sensorIndex}
									{@const data = chartData[key]}
									{@const maxTemp = Math.max(...data.map(d => d.value))}
									{@const minTemp = Math.min(...data.map(d => d.value))}
									{@const tempRange = maxTemp - minTemp || 1}
									
									<path
										d={data.map((point, i) => {
											const x = 50 + (i / (data.length - 1)) * 700;
											const y = 350 - ((point.value - minTemp) / tempRange) * 300;
											return `${i === 0 ? 'M' : 'L'} ${x} ${y}`;
										}).join(' ')}
										fill="none"
										stroke={getSensorColor(sensorIndex)}
										stroke-width="2"
									/>
									
									<!-- Data points -->
									{#each data as point, i}
										{@const x = 50 + (i / (data.length - 1)) * 700}
										{@const y = 350 - ((point.value - minTemp) / tempRange) * 300}
										<circle
											fill={getSensorColor(sensorIndex)}
											cx={x}
											cy={y}
											r="3"
										/>
									{/each}
								{/each}
								
								<!-- Legend -->
								{#each sensorKeys as key, i}
									<g transform="translate(50, {380 + i * 20})">
										<rect
											fill={getSensorColor(i)}
											x="0"
											y="0"
											width="15"
											height="15"
										/>
										<text x="20" y="12" font-size="12" fill="hsl(var(--foreground))">
											{key}
										</text>
									</g>
								{/each}
							</svg>
						</div>
					</CardContent>
				</Card>
			</TabsContent>
			
			<TabsContent value="heatmap">
				<Card>
					<CardHeader>
						<CardTitle>Temperature Heat Map</CardTitle>
						<CardDescription>Visualize temperature patterns across sensors and time</CardDescription>
					</CardHeader>
					<CardContent>
						<div class="text-center py-8">
							<Calendar class="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
							<p class="text-muted-foreground">Heat map visualization coming soon</p>
						</div>
					</CardContent>
				</Card>
			</TabsContent>
		</Tabs>
	{/if}
</div>