<script lang="ts">
	import { onMount } from 'svelte';
	import { alertClient, clientClient } from '$lib/grpc-client';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '$lib/components/ui/table';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Select from '$lib/components/ui/select';
	import { Switch } from '$lib/components/ui/switch';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import { Plus, Edit, Trash2, AlertTriangle, RefreshCw, Bell, BellOff } from '@lucide/svelte';
	import type { AlertRule, Alert as AlertInstance, AlertCondition, AlertAction } from '$lib/proto/jacuzzi/v1/alert/v1/alert_pb';
	import { AlertCondition_Operator, AlertAction_ActionType, AlertRuleSchema, AlertConditionSchema, AlertActionSchema } from '$lib/proto/jacuzzi/v1/alert/v1/alert_pb';
	import { create } from '@bufbuild/protobuf';
	import type { Client } from '$lib/proto/jacuzzi/v1/client/v1/client_pb';
	
	let rules = $state<AlertRule[]>([]);
	let alerts = $state<AlertInstance[]>([]);
	let clients = $state<Client[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let dialogOpen = $state(false);
	let deleteDialogOpen = $state(false);
	let selectedRule = $state<AlertRule | null>(null);
	let ruleToDelete = $state<AlertRule | null>(null);
	
	// Form state
	let ruleName = $state('');
	let ruleDescription = $state('');
	let ruleClientId = $state('');
	let ruleSensorId = $state('');
	let ruleSensorType = $state('');
	let ruleOperator = $state(String(AlertCondition_Operator.GREATER_THAN));
	let ruleThreshold = $state(70);
	let ruleDuration = $state(60);
	let ruleEnabled = $state(true);
	
	async function fetchAlertRules() {
		loading = true;
		error = null;
		
		try {
			const response = await alertClient.listAlertRules({
				clientId: '',
				enabledOnly: false,
				limit: 100,
				offset: 0
			});
			rules = response.rules;
		} catch (err) {
			console.error('Failed to fetch alert rules:', err);
			error = 'Failed to fetch alert rules';
		} finally {
			loading = false;
		}
	}
	
	async function fetchAlertHistory() {
		try {
			const response = await alertClient.getAlertHistory({
				ruleId: '',
				clientId: '',
				activeOnly: false,
				limit: 50
			});
			alerts = response.alerts;
		} catch (err) {
			console.error('Failed to fetch alert history:', err);
		}
	}
	
	async function fetchClients() {
		try {
			const response = await clientClient.listClients({
				onlineOnly: false,
				limit: 100,
				offset: 0
			});
			clients = response.clients;
		} catch (err) {
			console.error('Failed to fetch clients:', err);
		}
	}
	
	function resetForm() {
		ruleName = '';
		ruleDescription = '';
		ruleClientId = '';
		ruleSensorId = '';
		ruleSensorType = '';
		ruleOperator = String(AlertCondition_Operator.GREATER_THAN);
		ruleThreshold = 70;
		ruleDuration = 60;
		ruleEnabled = true;
		selectedRule = null;
	}
	
	function editRule(rule: AlertRule) {
		selectedRule = rule;
		ruleName = rule.name;
		ruleDescription = rule.description;
		ruleClientId = rule.clientId;
		ruleSensorId = rule.sensorId;
		ruleSensorType = rule.sensorType;
		ruleOperator = String(rule.condition?.operator || AlertCondition_Operator.GREATER_THAN);
		ruleThreshold = rule.condition?.threshold || 70;
		ruleDuration = rule.condition?.durationSeconds || 60;
		ruleEnabled = rule.enabled;
		dialogOpen = true;
	}
	
	async function saveRule() {
		try {
			const condition = create(AlertConditionSchema, {
				operator: Number(ruleOperator) as AlertCondition_Operator,
				threshold: ruleThreshold,
				durationSeconds: ruleDuration
			});
			
			const action = create(AlertActionSchema, {
				type: AlertAction_ActionType.LOG,
				config: {}
			});
			
			const rule = create(AlertRuleSchema, {
				id: selectedRule?.id || '',
				name: ruleName,
				description: ruleDescription,
				clientId: ruleClientId,
				sensorId: ruleSensorId,
				sensorType: ruleSensorType,
				condition: condition,
				actions: [action],
				enabled: ruleEnabled,
				createdAt: selectedRule?.createdAt,
				updatedAt: undefined
			});
			
			await alertClient.createAlertRule({ rule });
			
			dialogOpen = false;
			resetForm();
			await fetchAlertRules();
		} catch (err) {
			console.error('Failed to save alert rule:', err);
			error = 'Failed to save alert rule';
		}
	}
	
	async function deleteRule() {
		if (!ruleToDelete) return;
		
		try {
			await alertClient.deleteAlertRule({
				ruleId: ruleToDelete.id
			});
			
			deleteDialogOpen = false;
			ruleToDelete = null;
			await fetchAlertRules();
		} catch (err) {
			console.error('Failed to delete alert rule:', err);
			error = 'Failed to delete alert rule';
		}
	}
	
	function getOperatorLabel(operator: AlertCondition_Operator | string): string {
		const op = typeof operator === 'string' ? Number(operator) : operator;
		switch (op) {
			case AlertCondition_Operator.GREATER_THAN: return '>';
			case AlertCondition_Operator.LESS_THAN: return '<';
			case AlertCondition_Operator.EQUAL: return '=';
			case AlertCondition_Operator.NOT_EQUAL: return '≠';
			default: return '?';
		}
	}
	
	function formatTimestamp(timestamp: any): string {
		if (!timestamp) return 'Never';
		const date = timestamp.toDate ? timestamp.toDate() : new Date(timestamp);
		return date.toLocaleString();
	}
	
	onMount(() => {
		fetchAlertRules();
		fetchAlertHistory();
		fetchClients();
	});
</script>

<div class="max-w-7xl">
	<div class="mb-8">
		<h1 class="text-3xl font-bold mb-2">Alerts</h1>
		<p class="text-muted-foreground">Configure temperature alerts and notifications</p>
	</div>
	
	{#if error}
		<Alert class="mb-4" variant="destructive">
			<AlertDescription>{error}</AlertDescription>
		</Alert>
	{/if}
	
	<Tabs value="rules" class="space-y-4">
		<TabsList>
			<TabsTrigger value="rules">Alert Rules</TabsTrigger>
			<TabsTrigger value="history">Alert History</TabsTrigger>
		</TabsList>
		
		<TabsContent value="rules">
			<Card>
				<CardHeader>
					<div class="flex items-center justify-between">
						<div>
							<CardTitle>Alert Rules</CardTitle>
							<CardDescription>Configure conditions that trigger alerts</CardDescription>
						</div>
						<div class="flex gap-2">
							<Button variant="outline" size="sm" onclick={fetchAlertRules}>
								<RefreshCw class="h-4 w-4 mr-2" />
								Refresh
							</Button>
							<Button size="sm" onclick={() => { resetForm(); dialogOpen = true; }}>
								<Plus class="h-4 w-4 mr-2" />
								New Rule
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
					{:else if rules.length === 0}
						<div class="text-center py-8">
							<Bell class="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
							<p class="text-muted-foreground">No alert rules configured</p>
						</div>
					{:else}
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Name</TableHead>
									<TableHead>Target</TableHead>
									<TableHead>Condition</TableHead>
									<TableHead>Status</TableHead>
									<TableHead>Created</TableHead>
									<TableHead></TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								{#each rules as rule}
									<TableRow>
										<TableCell>
											<div>
												<p class="font-medium">{rule.name}</p>
												{#if rule.description}
													<p class="text-xs text-muted-foreground">{rule.description}</p>
												{/if}
											</div>
										</TableCell>
										<TableCell>
											<div class="text-sm">
												{#if rule.clientId}
													<p>Client: {rule.clientId}</p>
												{/if}
												{#if rule.sensorType}
													<p>Type: {rule.sensorType}</p>
												{/if}
												{#if rule.sensorId}
													<p class="text-xs text-muted-foreground">{rule.sensorId}</p>
												{/if}
											</div>
										</TableCell>
										<TableCell>
											<span class="font-mono text-sm">
												{getOperatorLabel(rule.condition?.operator || 0)} {rule.condition?.threshold}°C
												{#if rule.condition?.durationSeconds}
													<span class="text-muted-foreground">for {rule.condition.durationSeconds}s</span>
												{/if}
											</span>
										</TableCell>
										<TableCell>
											{#if rule.enabled}
												<Badge variant="default">Enabled</Badge>
											{:else}
												<Badge variant="secondary">Disabled</Badge>
											{/if}
										</TableCell>
										<TableCell>
											<span class="text-sm text-muted-foreground">
												{formatTimestamp(rule.createdAt)}
											</span>
										</TableCell>
										<TableCell>
											<div class="flex gap-1">
												<Button 
													variant="ghost" 
													size="sm"
													onclick={() => editRule(rule)}
												>
													<Edit class="h-4 w-4" />
												</Button>
												<Button 
													variant="ghost" 
													size="sm"
													onclick={() => { ruleToDelete = rule; deleteDialogOpen = true; }}
												>
													<Trash2 class="h-4 w-4" />
												</Button>
											</div>
										</TableCell>
									</TableRow>
								{/each}
							</TableBody>
						</Table>
					{/if}
				</CardContent>
			</Card>
		</TabsContent>
		
		<TabsContent value="history">
			<Card>
				<CardHeader>
					<CardTitle>Alert History</CardTitle>
					<CardDescription>Recent alerts that have been triggered</CardDescription>
				</CardHeader>
				<CardContent>
					{#if alerts.length === 0}
						<div class="text-center py-8">
							<BellOff class="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
							<p class="text-muted-foreground">No alerts triggered yet</p>
						</div>
					{:else}
						<div class="space-y-2">
							{#each alerts as alert}
								<div class="flex items-center justify-between p-3 border rounded-lg">
									<div class="flex items-center gap-3">
										<AlertTriangle class="h-5 w-5 {alert.isActive ? 'text-red-500' : 'text-muted-foreground'}" />
										<div>
											<p class="font-medium">{alert.message}</p>
											<p class="text-sm text-muted-foreground">
												{alert.clientId} • {alert.sensorId}
											</p>
										</div>
									</div>
									<div class="text-right">
										<p class="text-sm font-medium">{alert.value.toFixed(1)}°C</p>
										<p class="text-xs text-muted-foreground">
											{formatTimestamp(alert.triggeredAt)}
										</p>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</CardContent>
			</Card>
		</TabsContent>
	</Tabs>
</div>

<Dialog bind:open={dialogOpen}>
	<DialogContent class="max-w-lg">
		<DialogHeader>
			<DialogTitle>{selectedRule ? 'Edit' : 'Create'} Alert Rule</DialogTitle>
			<DialogDescription>
				Configure when to trigger temperature alerts
			</DialogDescription>
		</DialogHeader>
		
		<div class="space-y-4">
			<div>
				<Label for="name">Rule Name</Label>
				<Input id="name" bind:value={ruleName} placeholder="High CPU Temperature" />
			</div>
			
			<div>
				<Label for="description">Description</Label>
				<Textarea id="description" bind:value={ruleDescription} placeholder="Alert when CPU temperature is too high" />
			</div>
			
			<div class="grid grid-cols-2 gap-4">
				<div>
					<Label for="client">Client (optional)</Label>
					<Select.Root type="single" bind:value={ruleClientId}>
						<Select.Trigger id="client">
							{clients.find(c => c.id === ruleClientId)?.hostname || ruleClientId || "All clients"}
						</Select.Trigger>
						<Select.Content>
							<Select.Item value="" label="All clients">All clients</Select.Item>
							{#each clients as client}
								<Select.Item value={client.id} label={client.hostname}>{client.hostname}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>
				
				<div>
					<Label for="sensor-type">Sensor Type (optional)</Label>
					<Select.Root type="single" bind:value={ruleSensorType}>
						<Select.Trigger id="sensor-type">
							{ruleSensorType || "All types"}
						</Select.Trigger>
						<Select.Content>
							<Select.Item value="" label="All types">All types</Select.Item>
							<Select.Item value="CPU" label="CPU">CPU</Select.Item>
							<Select.Item value="GPU" label="GPU">GPU</Select.Item>
							<Select.Item value="DISK" label="Disk">Disk</Select.Item>
						</Select.Content>
					</Select.Root>
				</div>
			</div>
			
			<div>
				<Label for="sensor-id">Sensor ID (optional)</Label>
				<Input id="sensor-id" bind:value={ruleSensorId} placeholder="Leave empty for all sensors" />
			</div>
			
			<div class="space-y-2">
				<Label>Condition</Label>
				<div class="flex gap-2">
					<Select.Root type="single" bind:value={ruleOperator}>
						<Select.Trigger class="w-32">
							{getOperatorLabel(ruleOperator)}
						</Select.Trigger>
						<Select.Content>
							<Select.Item value={String(AlertCondition_Operator.GREATER_THAN)} label="Greater than">Greater than</Select.Item>
							<Select.Item value={String(AlertCondition_Operator.LESS_THAN)} label="Less than">Less than</Select.Item>
							<Select.Item value={String(AlertCondition_Operator.EQUAL)} label="Equal to">Equal to</Select.Item>
							<Select.Item value={String(AlertCondition_Operator.NOT_EQUAL)} label="Not equal to">Not equal to</Select.Item>
						</Select.Content>
					</Select.Root>
					<Input type="number" bind:value={ruleThreshold} class="w-24" />
					<span class="flex items-center">°C</span>
				</div>
				<div class="flex items-center gap-2">
					<Label for="duration" class="text-sm">For at least</Label>
					<Input id="duration" type="number" bind:value={ruleDuration} class="w-24" />
					<span class="text-sm">seconds</span>
				</div>
			</div>
			
			<div class="flex items-center justify-between">
				<Label for="enabled">Enable Rule</Label>
				<Switch id="enabled" bind:checked={ruleEnabled} />
			</div>
		</div>
		
		<DialogFooter>
			<Button variant="outline" onclick={() => { dialogOpen = false; resetForm(); }}>
				Cancel
			</Button>
			<Button onclick={saveRule} disabled={!ruleName}>
				{selectedRule ? 'Update' : 'Create'} Rule
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>

<Dialog bind:open={deleteDialogOpen}>
	<DialogContent>
		<DialogHeader>
			<DialogTitle>Delete Alert Rule</DialogTitle>
			<DialogDescription>
				Are you sure you want to delete the rule "{ruleToDelete?.name}"? This action cannot be undone.
			</DialogDescription>
		</DialogHeader>
		<DialogFooter>
			<Button variant="outline" onclick={() => { deleteDialogOpen = false; ruleToDelete = null; }}>
				Cancel
			</Button>
			<Button variant="destructive" onclick={deleteRule}>
				Delete Rule
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>