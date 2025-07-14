<script lang="ts">
	import { onMount } from 'svelte';
	import { clientClient } from '$lib/grpc-client';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '$lib/components/ui/table';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '$lib/components/ui/dialog';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Switch } from '$lib/components/ui/switch';
	import { Monitor, Info, RefreshCw, Plus } from '@lucide/svelte';
	import type { Client, SensorInfo } from '$lib/proto/jacuzzi/v1/client/v1/client_pb';
	
	let clients: Client[] = [];
	let loading = false;
	let error: string | null = null;
	let selectedClient: Client | null = null;
	let clientSensors: SensorInfo[] = [];
	let showOnlineOnly = false;
	let dialogOpen = false;
	
	// Metadata form
	let metadataKey = '';
	let metadataValue = '';
	
	async function fetchClients() {
		loading = true;
		error = null;
		
		try {
			const response = await clientClient.listClients({
				onlineOnly: showOnlineOnly,
				limit: 100,
				offset: 0
			});
			clients = response.clients;
		} catch (err) {
			console.error('Failed to fetch clients:', err);
			error = 'Failed to fetch clients';
		} finally {
			loading = false;
		}
	}
	
	async function fetchClientDetails(clientId: string) {
		try {
			const response = await clientClient.getClient({
				clientId
			});
			selectedClient = response.client || null;
			clientSensors = response.sensors;
			dialogOpen = true;
		} catch (err) {
			console.error('Failed to fetch client details:', err);
			error = 'Failed to fetch client details';
		}
	}
	
	async function updateClientMetadata() {
		if (!selectedClient || !metadataKey || !metadataValue) return;
		
		try {
			const newMetadata = { ...selectedClient.metadata };
			newMetadata[metadataKey] = metadataValue;
			
			await clientClient.updateClient({
				clientId: selectedClient.id,
				metadata: newMetadata
			});
			
			// Refresh client details
			await fetchClientDetails(selectedClient.id);
			metadataKey = '';
			metadataValue = '';
		} catch (err) {
			console.error('Failed to update client:', err);
			error = 'Failed to update client metadata';
		}
	}
	
	function formatTimestamp(timestamp: any): string {
		if (!timestamp) return 'Never';
		const date = timestamp.toDate ? timestamp.toDate() : new Date(timestamp);
		return date.toLocaleString();
	}
	
	function getRelativeTime(timestamp: any): string {
		if (!timestamp) return 'Never';
		const date = timestamp.toDate ? timestamp.toDate() : new Date(timestamp);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(diff / 3600000);
		const days = Math.floor(diff / 86400000);
		
		if (minutes < 1) return 'Just now';
		if (minutes < 60) return `${minutes}m ago`;
		if (hours < 24) return `${hours}h ago`;
		return `${days}d ago`;
	}
	
	function getSensorTypeIcon(type: string): string {
		switch (type?.toUpperCase()) {
			case 'CPU': return '🖥️';
			case 'GPU': return '🎮';
			case 'DISK': return '💾';
			default: return '🌡️';
		}
	}
	
	onMount(() => {
		fetchClients();
	});
	
	$: if (showOnlineOnly !== undefined) {
		fetchClients();
	}
</script>

<div class="max-w-7xl">
	<div class="mb-8">
		<h1 class="text-3xl font-bold mb-2">Clients</h1>
		<p class="text-muted-foreground">Manage and monitor connected clients</p>
	</div>
	
	{#if error}
		<Alert class="mb-4" variant="destructive">
			<AlertDescription>{error}</AlertDescription>
		</Alert>
	{/if}
	
	<Card>
		<CardHeader>
			<div class="flex items-center justify-between">
				<div>
					<CardTitle>Connected Clients</CardTitle>
					<CardDescription>View all clients reporting temperature data</CardDescription>
				</div>
				<div class="flex items-center gap-4">
					<div class="flex items-center gap-2">
						<Switch id="online-only" bind:checked={showOnlineOnly} />
						<Label for="online-only">Online only</Label>
					</div>
					<Button variant="outline" size="sm" onclick={fetchClients}>
						<RefreshCw class="h-4 w-4 mr-2" />
						Refresh
					</Button>
				</div>
			</div>
		</CardHeader>
		<CardContent>
			{#if loading}
				<div class="space-y-2">
					{#each [1, 2, 3] as i}
						<Skeleton class="h-12 w-full" />
					{/each}
				</div>
			{:else if clients.length === 0}
				<div class="text-center py-8">
					<Monitor class="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
					<p class="text-muted-foreground">No clients found</p>
				</div>
			{:else}
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Hostname</TableHead>
							<TableHead>IP Address</TableHead>
							<TableHead>OS / Arch</TableHead>
							<TableHead>Status</TableHead>
							<TableHead>Last Seen</TableHead>
							<TableHead>First Seen</TableHead>
							<TableHead></TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{#each clients as client}
							<TableRow>
								<TableCell class="font-medium">{client.hostname}</TableCell>
								<TableCell>{client.ipAddress}</TableCell>
								<TableCell>{client.os} / {client.arch}</TableCell>
								<TableCell>
									{#if client.isOnline}
										<Badge variant="default">Online</Badge>
									{:else}
										<Badge variant="secondary">Offline</Badge>
									{/if}
								</TableCell>
								<TableCell>
									<span class="text-sm text-muted-foreground">
										{getRelativeTime(client.lastSeen)}
									</span>
								</TableCell>
								<TableCell>
									<span class="text-sm text-muted-foreground">
										{formatTimestamp(client.firstSeen)}
									</span>
								</TableCell>
								<TableCell>
									<Button 
										variant="ghost" 
										size="sm"
										onclick={() => fetchClientDetails(client.id)}
									>
										<Info class="h-4 w-4" />
									</Button>
								</TableCell>
							</TableRow>
						{/each}
					</TableBody>
				</Table>
			{/if}
		</CardContent>
	</Card>
</div>

<Dialog bind:open={dialogOpen}>
	<DialogContent class="max-w-2xl">
		<DialogHeader>
			<DialogTitle>Client Details</DialogTitle>
			<DialogDescription>
				{selectedClient?.hostname} ({selectedClient?.id})
			</DialogDescription>
		</DialogHeader>
		
		{#if selectedClient}
			<div class="space-y-4">
				<div class="grid grid-cols-2 gap-4">
					<div>
						<Label class="text-sm text-muted-foreground">IP Address</Label>
						<p class="font-medium">{selectedClient.ipAddress}</p>
					</div>
					<div>
						<Label class="text-sm text-muted-foreground">Operating System</Label>
						<p class="font-medium">{selectedClient.os} / {selectedClient.arch}</p>
					</div>
					<div>
						<Label class="text-sm text-muted-foreground">Status</Label>
						<p class="font-medium">
							{#if selectedClient.isOnline}
								<Badge variant="default">Online</Badge>
							{:else}
								<Badge variant="secondary">Offline</Badge>
							{/if}
						</p>
					</div>
					<div>
						<Label class="text-sm text-muted-foreground">Last Seen</Label>
						<p class="font-medium">{formatTimestamp(selectedClient.lastSeen)}</p>
					</div>
				</div>
				
				{#if clientSensors.length > 0}
					<div>
						<Label class="text-sm text-muted-foreground mb-2">Active Sensors</Label>
						<div class="space-y-2">
							{#each clientSensors as sensor}
								<div class="flex items-center justify-between p-2 bg-muted/50 rounded">
									<div class="flex items-center gap-2">
										<span>{getSensorTypeIcon(sensor.sensorType)}</span>
										<div>
											<p class="font-medium">{sensor.sensorName}</p>
											<p class="text-xs text-muted-foreground">{sensor.sensorId}</p>
										</div>
									</div>
									<div class="text-right">
										<p class="font-bold">{sensor.currentTemperature.toFixed(1)}°C</p>
										<p class="text-xs text-muted-foreground">
											{getRelativeTime(sensor.lastReading)}
										</p>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}
				
				<div>
					<Label class="text-sm text-muted-foreground mb-2">Metadata</Label>
					{#if Object.keys(selectedClient.metadata || {}).length > 0}
						<div class="space-y-1">
							{#each Object.entries(selectedClient.metadata) as [key, value]}
								<div class="flex justify-between text-sm">
									<span class="font-medium">{key}:</span>
									<span>{value}</span>
								</div>
							{/each}
						</div>
					{:else}
						<p class="text-sm text-muted-foreground">No metadata</p>
					{/if}
					
					<div class="flex gap-2 mt-3">
						<Input 
							placeholder="Key" 
							bind:value={metadataKey}
							class="flex-1"
						/>
						<Input 
							placeholder="Value" 
							bind:value={metadataValue}
							class="flex-1"
						/>
						<Button 
							size="sm"
							onclick={updateClientMetadata}
							disabled={!metadataKey || !metadataValue}
						>
							<Plus class="h-4 w-4" />
						</Button>
					</div>
				</div>
			</div>
		{/if}
		
		<DialogFooter>
			<Button variant="outline" onclick={() => dialogOpen = false}>
				Close
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>