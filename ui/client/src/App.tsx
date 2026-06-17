import { Switch, Route } from "wouter";
import { queryClient } from "./lib/queryClient";
import { QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/toaster";
import { TooltipProvider } from "@/components/ui/tooltip";
import NotFound from "@/pages/not-found";
import Dashboard from "@/pages/dashboard";
import Configuration from "@/pages/configuration";
import Memory from "@/pages/memory";
import SystemLogs from "@/pages/logs";
import Projects from "@/pages/projects";
import Patterns from "@/pages/patterns";
import SearchPage from "@/pages/search";
import Timeline from "@/pages/timeline";
import UsagePage from "@/pages/usage";

function Router() {
  return (
    <Switch>
      <Route path="/" component={Dashboard} />
      <Route path="/projects" component={Projects} />
      <Route path="/memory" component={Memory} />
      <Route path="/search" component={SearchPage} />
      <Route path="/patterns" component={Patterns} />
      <Route path="/timeline" component={Timeline} />
      <Route path="/usage" component={UsagePage} />
      <Route path="/logs" component={SystemLogs} />
      <Route path="/config" component={Configuration} />
      <Route component={NotFound} />
    </Switch>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <Toaster />
        <Router />
      </TooltipProvider>
    </QueryClientProvider>
  );
}

export default App;
