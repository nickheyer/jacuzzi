<script lang="ts">
	import { onMount } from 'svelte';
	import { settingsClient } from '$lib/grpc-client';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Select from '$lib/components/ui/select';
	import { Switch } from '$lib/components/ui/switch';
	import { Separator } from '$lib/components/ui/separator';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import { Save, RefreshCw, Settings2, Mail, Database, Bell } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import type { Settings, EmailSettings } from '$lib/proto/jacuzzi/v1/settings/v1/settings_pb';
	import { SettingsSchema, EmailSettingsSchema } from '$lib/proto/jacuzzi/v1/settings/v1/settings_pb';
	import { create } from '@bufbuild/protobuf';
	
	let settings = $state<Settings | null>(null);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let saving = $state(false);
	
	// Form state - initialized with defaults
	let siteName = $state('Jacuzzi Monitor');
	let timezone = $state('UTC');
	let retentionDays = $state(30);
	let aggregationInterval = $state(300);
	let temperatureUnit = $state('celsius');
	let theme = $state('system');
	let alertsEnabled = $state(true);
	let alertCheckInterval = $state(60);
	let maxConcurrentClients = $state(100);
	let apiRateLimit = $state(1000);
	
	// Email settings
	let smtpHost = $state('');
	let smtpPort = $state(587);
	let smtpUsername = $state('');
	let smtpPassword = $state('');
	let useTls = $state(true);
	let fromAddress = $state('');
	let adminEmails = $state<string[]>([]);
	let newAdminEmail = $state('');
	
	async function fetchSettings() {
		loading = true;
		error = null;
		
		try {
			const response = await settingsClient.getSettings({});
			settings = response.settings || null;
			
			if (settings) {
				// Update form state with fetched settings
				siteName = settings.siteName || 'Jacuzzi Monitor';
				timezone = settings.timezone || 'UTC';
				retentionDays = settings.retentionDays || 30;
				aggregationInterval = settings.aggregationIntervalSeconds || 300;
				temperatureUnit = settings.temperatureUnit || 'celsius';
				theme = settings.theme || 'system';
				alertsEnabled = settings.alertsEnabled ?? true;
				alertCheckInterval = settings.alertCheckIntervalSeconds || 60;
				maxConcurrentClients = settings.maxConcurrentClients || 100;
				apiRateLimit = settings.apiRateLimit || 1000;
				
				// Email settings
				if (settings.emailSettings) {
					smtpHost = settings.emailSettings.smtpHost || '';
					smtpPort = settings.emailSettings.smtpPort || 587;
					smtpUsername = settings.emailSettings.smtpUsername || '';
					smtpPassword = settings.emailSettings.smtpPassword || '';
					useTls = settings.emailSettings.useTls ?? true;
					fromAddress = settings.emailSettings.fromAddress || '';
					adminEmails = settings.emailSettings.adminEmails || [];
				}
			}
		} catch (err) {
			console.error('Failed to fetch settings:', err);
			error = 'Failed to fetch settings';
		} finally {
			loading = false;
		}
	}
	
	async function saveSettings() {
		saving = true;
		error = null;
		
		try {
			const emailSettings = create(EmailSettingsSchema, {
				smtpHost,
				smtpPort,
				smtpUsername,
				smtpPassword,
				useTls,
				fromAddress,
				adminEmails
			});
			
			const updatedSettings = create(SettingsSchema, {
				siteName,
				timezone,
				retentionDays,
				aggregationIntervalSeconds: aggregationInterval,
				temperatureUnit,
				theme,
				alertsEnabled,
				alertCheckIntervalSeconds: alertCheckInterval,
				emailSettings,
				maxConcurrentClients,
				apiRateLimit
			});
			
			await settingsClient.updateSettings({
				settings: updatedSettings
			});
			
			toast.success('Settings saved successfully');
		} catch (err) {
			console.error('Failed to save settings:', err);
			error = 'Failed to save settings';
			toast.error('Failed to save settings');
		} finally {
			saving = false;
		}
	}
	
	function addAdminEmail() {
		if (newAdminEmail && !adminEmails.includes(newAdminEmail)) {
			adminEmails = [...adminEmails, newAdminEmail];
			newAdminEmail = '';
		}
	}
	
	function removeAdminEmail(email: string) {
		adminEmails = adminEmails.filter(e => e !== email);
	}
	
	const timezones = [
		'UTC',
		'America/New_York',
		'America/Chicago',
		'America/Denver',
		'America/Los_Angeles',
		'Europe/London',
		'Europe/Paris',
		'Europe/Berlin',
		'Asia/Tokyo',
		'Asia/Shanghai',
		'Australia/Sydney'
	];
	
	onMount(() => {
		fetchSettings();
	});
</script>

<div class="max-w-4xl">
	<div class="mb-8">
		<h1 class="text-3xl font-bold mb-2">Settings</h1>
		<p class="text-muted-foreground">Configure system-wide settings and preferences</p>
	</div>
	
	{#if error}
		<Alert class="mb-4" variant="destructive">
			<AlertDescription>{error}</AlertDescription>
		</Alert>
	{/if}
	
	{#if loading}
		<Card>
			<CardContent class="py-8">
				<div class="flex items-center justify-center">
					<RefreshCw class="h-6 w-6 animate-spin" />
				</div>
			</CardContent>
		</Card>
	{:else}
		<form onsubmit={(e) => { e.preventDefault(); saveSettings(); }}>
			<Tabs value="general" class="space-y-4">
				<TabsList class="grid w-full grid-cols-4">
					<TabsTrigger value="general">General</TabsTrigger>
					<TabsTrigger value="display">Display</TabsTrigger>
					<TabsTrigger value="alerts">Alerts</TabsTrigger>
					<TabsTrigger value="performance">Performance</TabsTrigger>
				</TabsList>
				
				<TabsContent value="general">
					<Card>
						<CardHeader>
							<CardTitle>General Settings</CardTitle>
							<CardDescription>Basic system configuration</CardDescription>
						</CardHeader>
						<CardContent class="space-y-4">
							<div class="space-y-2">
								<Label for="site-name">Site Name</Label>
								<Input id="site-name" bind:value={siteName} placeholder="Jacuzzi Monitor" />
							</div>
							
							<div class="space-y-2">
								<Label for="timezone">Timezone</Label>
								<Select.Root type="single" bind:value={timezone}>
									<Select.Trigger id="timezone">
										{timezone}
									</Select.Trigger>
									<Select.Content>
										{#each timezones as tz}
											<Select.Item value={tz} label={tz}>{tz}</Select.Item>
										{/each}
									</Select.Content>
								</Select.Root>
							</div>
							
							<Separator />
							
							<div>
								<h3 class="text-lg font-semibold mb-4 flex items-center gap-2">
									<Database class="h-5 w-5" />
									Data Retention
								</h3>
								
								<div class="space-y-4">
									<div class="space-y-2">
										<Label for="retention">Retention Period (days)</Label>
										<Input id="retention" type="number" bind:value={retentionDays} min="1" max="365" />
										<p class="text-sm text-muted-foreground">How long to keep temperature data</p>
									</div>
									
									<div class="space-y-2">
										<Label for="aggregation">Aggregation Interval (seconds)</Label>
										<Input id="aggregation" type="number" bind:value={aggregationInterval} min="60" max="3600" />
										<p class="text-sm text-muted-foreground">Interval for data aggregation</p>
									</div>
								</div>
							</div>
						</CardContent>
					</Card>
				</TabsContent>
				
				<TabsContent value="display">
					<Card>
						<CardHeader>
							<CardTitle>Display Settings</CardTitle>
							<CardDescription>Customize the user interface</CardDescription>
						</CardHeader>
						<CardContent class="space-y-4">
							<div class="space-y-2">
								<Label for="temp-unit">Temperature Unit</Label>
								<Select.Root type="single" bind:value={temperatureUnit}>
									<Select.Trigger id="temp-unit">
										{temperatureUnit === 'celsius' ? 'Celsius (°C)' : 'Fahrenheit (°F)'}
									</Select.Trigger>
									<Select.Content>
										<Select.Item value="celsius" label="Celsius (°C)">Celsius (°C)</Select.Item>
										<Select.Item value="fahrenheit" label="Fahrenheit (°F)">Fahrenheit (°F)</Select.Item>
									</Select.Content>
								</Select.Root>
							</div>
							
							<div class="space-y-2">
								<Label for="theme">Theme</Label>
								<Select.Root type="single" bind:value={theme}>
									<Select.Trigger id="theme">
										{theme.charAt(0).toUpperCase() + theme.slice(1)}
									</Select.Trigger>
									<Select.Content>
										<Select.Item value="light" label="Light">Light</Select.Item>
										<Select.Item value="dark" label="Dark">Dark</Select.Item>
										<Select.Item value="system" label="System">System</Select.Item>
									</Select.Content>
								</Select.Root>
							</div>
						</CardContent>
					</Card>
				</TabsContent>
				
				<TabsContent value="alerts">
					<Card>
						<CardHeader>
							<CardTitle>Alert Settings</CardTitle>
							<CardDescription>Configure alert system and notifications</CardDescription>
						</CardHeader>
						<CardContent class="space-y-4">
							<div class="flex items-center justify-between">
								<div class="space-y-0.5">
									<Label for="alerts-enabled">Enable Alerts</Label>
									<p class="text-sm text-muted-foreground">Process and send temperature alerts</p>
								</div>
								<Switch id="alerts-enabled" bind:checked={alertsEnabled} />
							</div>
							
							<div class="space-y-2">
								<Label for="alert-interval">Alert Check Interval (seconds)</Label>
								<Input id="alert-interval" type="number" bind:value={alertCheckInterval} min="10" max="600" disabled={!alertsEnabled} />
							</div>
							
							<Separator />
							
							<div>
								<h3 class="text-lg font-semibold mb-4 flex items-center gap-2">
									<Mail class="h-5 w-5" />
									Email Configuration
								</h3>
								
								<div class="space-y-4">
									<div class="grid grid-cols-2 gap-4">
										<div class="space-y-2">
											<Label for="smtp-host">SMTP Host</Label>
											<Input id="smtp-host" bind:value={smtpHost} placeholder="smtp.example.com" />
										</div>
										<div class="space-y-2">
											<Label for="smtp-port">SMTP Port</Label>
											<Input id="smtp-port" type="number" bind:value={smtpPort} />
										</div>
									</div>
									
									<div class="grid grid-cols-2 gap-4">
										<div class="space-y-2">
											<Label for="smtp-user">SMTP Username</Label>
											<Input id="smtp-user" bind:value={smtpUsername} />
										</div>
										<div class="space-y-2">
											<Label for="smtp-pass">SMTP Password</Label>
											<Input id="smtp-pass" type="password" bind:value={smtpPassword} />
										</div>
									</div>
									
									<div class="flex items-center justify-between">
										<Label for="use-tls">Use TLS</Label>
										<Switch id="use-tls" bind:checked={useTls} />
									</div>
									
									<div class="space-y-2">
										<Label for="from-address">From Address</Label>
										<Input id="from-address" type="email" bind:value={fromAddress} placeholder="alerts@example.com" />
									</div>
									
									<div class="space-y-2">
										<Label>Admin Email Addresses</Label>
										<div class="space-y-2">
											{#each adminEmails as email}
												<div class="flex items-center gap-2">
													<Input value={email} disabled class="flex-1" />
													<Button 
														type="button"
														variant="outline" 
														size="sm"
														onclick={() => removeAdminEmail(email)}
													>
														Remove
													</Button>
												</div>
											{/each}
											<div class="flex gap-2">
												<Input 
													placeholder="admin@example.com" 
													bind:value={newAdminEmail}
													onkeydown={(e) => e.key === 'Enter' && (e.preventDefault(), addAdminEmail())}
												/>
												<Button 
													type="button"
													variant="outline"
													onclick={addAdminEmail}
													disabled={!newAdminEmail}
												>
													Add
												</Button>
											</div>
										</div>
									</div>
								</div>
							</div>
						</CardContent>
					</Card>
				</TabsContent>
				
				<TabsContent value="performance">
					<Card>
						<CardHeader>
							<CardTitle>Performance Settings</CardTitle>
							<CardDescription>System limits and optimization</CardDescription>
						</CardHeader>
						<CardContent class="space-y-4">
							<div class="space-y-2">
								<Label for="max-clients">Max Concurrent Clients</Label>
								<Input id="max-clients" type="number" bind:value={maxConcurrentClients} min="1" max="1000" />
								<p class="text-sm text-muted-foreground">Maximum number of clients that can connect simultaneously</p>
							</div>
							
							<div class="space-y-2">
								<Label for="rate-limit">API Rate Limit (requests/minute)</Label>
								<Input id="rate-limit" type="number" bind:value={apiRateLimit} min="10" max="10000" />
								<p class="text-sm text-muted-foreground">Maximum API requests per minute per client</p>
							</div>
						</CardContent>
					</Card>
				</TabsContent>
			</Tabs>
			
			<div class="flex justify-end gap-2 mt-6">
				<Button type="button" variant="outline" onclick={fetchSettings}>
					<RefreshCw class="h-4 w-4 mr-2" />
					Reset
				</Button>
				<Button type="submit" disabled={saving}>
					<Save class="h-4 w-4 mr-2" />
					{saving ? 'Saving...' : 'Save Settings'}
				</Button>
			</div>
		</form>
	{/if}
</div>