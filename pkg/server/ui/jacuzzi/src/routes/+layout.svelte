<script lang="ts">
	import '../app.css';
	import { ModeWatcher } from 'mode-watcher';
	import { SidebarProvider, Sidebar, SidebarContent, SidebarGroup, SidebarGroupContent, SidebarMenu, SidebarMenuItem, SidebarMenuButton, SidebarHeader, SidebarFooter, SidebarTrigger } from '$lib/components/ui/sidebar';
	import { Home, Users, AlertCircle, Settings, Thermometer, Activity } from '@lucide/svelte';
	import { page } from '$app/stores';
	
	let { children } = $props();
	
	const menuItems = [
		{ title: 'Dashboard', icon: Home, href: '/' },
		{ title: 'Clients', icon: Users, href: '/clients' },
		{ title: 'Alerts', icon: AlertCircle, href: '/alerts' },
		{ title: 'Statistics', icon: Activity, href: '/stats' },
		{ title: 'Settings', icon: Settings, href: '/settings' }
	];
</script>

<ModeWatcher />

<SidebarProvider>
	<div class="flex h-screen w-full">
		<Sidebar>
			<SidebarHeader class="p-4">
				<div class="flex items-center gap-2">
					<Thermometer class="h-6 w-6" />
					<span class="text-lg font-semibold">Jacuzzi Monitor</span>
				</div>
			</SidebarHeader>
			
			<SidebarContent>
				<SidebarGroup>
					<SidebarGroupContent>
						<SidebarMenu>
							{#each menuItems as item}
								{@const Icon = item.icon}
								<SidebarMenuItem>
									<a href={item.href} class="w-full">
										<SidebarMenuButton isActive={$page.url.pathname === item.href}>
											<Icon class="h-4 w-4" />
											<span>{item.title}</span>
										</SidebarMenuButton>
									</a>
								</SidebarMenuItem>
							{/each}
						</SidebarMenu>
					</SidebarGroupContent>
				</SidebarGroup>
			</SidebarContent>
			
			<SidebarFooter class="p-4">
				<div class="text-xs text-muted-foreground">
					Hardware Temperature Monitor
				</div>
			</SidebarFooter>
		</Sidebar>
		
		<main class="flex-1 overflow-auto">
			<div class="sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-b px-4 py-2">
				<SidebarTrigger />
			</div>
			<div class="p-4">
				{@render children()}
			</div>
		</main>
	</div>
</SidebarProvider>
