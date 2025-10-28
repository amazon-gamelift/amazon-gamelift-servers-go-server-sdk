- [Go Server SDK for Amazon GameLift Servers Metrics API](#go-server-sdk-for-amazon-gamelift-servers-metrics-api)
  - [Metrics Setup & Workflow](#metrics-setup--workflow)
    - [Step 1: Initialize the Metrics System](#step-1-initialize-the-metrics-system)
    - [Step 2: Create and Use Metrics](#step-2-create-and-use-metrics)
    - [Example Usage Patterns](#example-usage-patterns)
  - [Metrics Usage & Operations](#metrics-usage--operations)
    - [Gauges](#gauges)
    - [Counters](#counters)
    - [Timers](#timers)
    - [Derived Metrics](#derived-metrics)
    - [Samplers](#samplers)
    - [Tagging](#tagging)
  - [Appendix](#appendix)
    - [Choosing the Right Metric Type](#choosing-the-right-metric-type)

# Go Server SDK for Amazon GameLift Servers Metrics API

The Go server SDK for Amazon GameLift Servers provides a comprehensive metrics system for collecting and sending custom
metrics from your game servers hosted on Amazon GameLift Server. These metrics can be integrated with various visualization
and aggregation tools including Amazon Managed Grafana, Amazon Managed Prometheus, Amazon CloudWatch, and other monitoring platforms.
This documentation provides explanations and instructions for advanced configuration and custom metrics implementation.
For a quick start and workflow setup, refer to [METRICS.md](METRICS.md).

## Metrics Setup & Workflow

### Step 1: Initialize the Metrics System

Initialize the metrics system once per application using one of two approaches:

#### Option 1: Initialize Metrics with Default Configuration

The simple approach is to use the server package with default configuration by calling `InitMetricsFromEnvironment()`.
This method configures metrics using default environment variables. See the example below for initializing the metrics system:

```golang
import (
    "log"

    sdkModule "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server"
)

func main() {
    // Initialize metrics system with environment-based configuration
    metrics, err := sdkModule.InitMetricsFromEnvironment()
    if err != nil {
        log.Printf("Failed to initialize metrics: %v", err)
        return
    }

    // Continue with metrics creation...
}
```

**Metrics Environment Variables**:
- **GAMELIFT_STATSD_HOST** - StatsD server host (default: localhost)
- **GAMELIFT_STATSD_PORT** - StatsD server port (default: 8125)
- **GAMELIFT_CRASH_REPORTER_HOST** - Crash reporter host (default: localhost)
- **GAMELIFT_CRASH_REPORTER_PORT** - Crash reporter port (default: 8126)
- **GAMELIFT_FLUSH_INTERVAL_MS** - Metrics flush interval in milliseconds (default: 10000)
- **GAMELIFT_MAX_PACKET_SIZE** - Maximum packet size in bytes (default: 512)

#### Option 2: Initialize Metrics with Custom Configuration

For applications requiring specific configuration parameters, use `InitMetrics()` with `MetricsParameters`:

```golang
import (
    "log"

    sdkModule "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server"
)

func main() {
    // Define custom metrics configuration parameters
    params := sdkModule.MetricsParameters{
        StatsdHost:        "localhost",  // StatsD server hostname
        StatsdPort:        8125,         // StatsD server port
        CrashReporterHost: "localhost",  // Crash reporter hostname
        CrashReporterPort: 8126,         // Crash reporter port
        FlushIntervalMs:   5000,         // Flush metrics every 5 seconds
        MaxPacketSize:     1024,         // Maximum UDP packet size (1KB)
    }

    // Initialize metrics system with custom parameters
    metrics, err := sdkModule.InitMetrics(params)
    if err != nil {
        log.Printf("Failed to initialize metrics: %v", err)
        return
    }

    // Continue with metrics creation...
}
```

### Step 2: Create and Use Metrics

```golang
// Create a counter metric to track server bytes received
serverBytesInCounter, err := metrics.Counter("server_bytes_in")
if err != nil {
    log.Printf("Failed to create serverBytesInCounter: %v", err)
    return
}

// Create a gauge metric to track current players
serverPlayersGauge, err := metrics.Gauge("server_players")
if err != nil {
    log.Printf("Failed to create serverPlayersGauge: %v", err)
    return
}

// Create a timer metric to track server tick duration
serverTickTimeTimer, err := metrics.Timer("server_tick_time")
if err != nil {
    log.Printf("Failed to create serverTickTimeTimer: %v", err)
    return
}

// Use metrics throughout your application
serverBytesInCounter.Increment()             // Count events
serverPlayersGauge.Set(42)                   // Track current values
serverTickTimeTimer.SetMilliseconds(125)     // Record timing (milliseconds)
```


### Example Usage Patterns

```golang
package main

import (
    "log"
    "time"

    "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics"
    sdkModule "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server"
)

type GameServer struct {
    playerJoinsCounter *metrics.Counter
    activePlayersGauge *metrics.Gauge
    matchDurationTimer *metrics.Timer
}

func NewGameServer() (*GameServer, error) {
    // Initialize metrics with default configuration
    metrics, err := sdkModule.InitMetricsFromEnvironment()
    if err != nil {
        return nil, err
    }

    // Create metrics once during initialization
    playerJoinsCounter, err := metrics.Counter("server_player_joins")
    if err != nil {
        return nil, err
    }

    activePlayersGauge, err := metrics.Gauge("server_players")
    if err != nil {
        return nil, err
    }

    matchDurationTimer, err := metrics.Timer("match_duration")
    if err != nil {
        return nil, err
    }

    return &GameServer{
        playerJoinsCounter: playerJoinsCounter,
        activePlayersGauge: activePlayersGauge,
        matchDurationTimer: matchDurationTimer,
    }, nil
}

func (gs *GameServer) OnPlayerJoin() {
    gs.playerJoinsCounter.Increment()
    gs.activePlayersGauge.Increment()
}

func (gs *GameServer) OnPlayerLeave() {
    gs.activePlayersGauge.Decrement()
}

func (gs *GameServer) OnMatchEnd(duration time.Duration) {
    gs.matchDurationTimer.SetMilliseconds(float64(duration.Milliseconds()))
}

func (gs *GameServer) UpdatePlayerCount(count int) {
    gs.activePlayersGauge.Set(count)
}
```


## Metrics Usage & Operations
For detailed code reference, see [metric.go](../metrics/model/metric.go).

### Gauges
Gauges represent metrics that track the current value of something over time. They maintain state and are ideal for
measurements like player count, memory usage, connection count, or any value that can go up and down.

**When to use:** For values that can go up and down, representing current state or levels.

**Key characteristics:**
- Values can increase, decrease, or be set to specific values
- Represent current state snapshots
- Used for monitoring capacity, load, and current levels
- Answer "How much right now?" and "What's the current state?"

```golang
gauge, err := metrics.Gauge("server_players")
if err != nil {
    log.Fatal(err)
}
gauge.Set(42)              // Set to exact value
gauge.Increment()          // Add 1
gauge.Decrement()          // Subtract 1
gauge.Add(5)               // Add 5
gauge.Subtract(3)          // Subtract 3
gauge.Reset()              // Reset to 0
```

**Common use cases:**
```golang
// Player and session tracking
activePlayersGauge, err := metrics.Gauge("server_players")
if err != nil {
    log.Fatal(err)
}
queuedPlayersGauge, err := metrics.Gauge("queued_players")
if err != nil {
    log.Fatal(err)
}
activeMatchesGauge, err := metrics.Gauge("active_matches")
if err != nil {
    log.Fatal(err)
}

// Resource monitoring
cpuUsageGauge, err := metrics.Gauge("cpu_usage_percent")
if err != nil {
    log.Fatal(err)
}
memoryUsageGauge, err := metrics.Gauge("memory_usage_mb")
if err != nil {
    log.Fatal(err)
}
diskSpaceGauge, err := metrics.Gauge("disk_space_gb")
if err != nil {
    log.Fatal(err)
}

// Game state tracking
healthGauge, err := metrics.Gauge("player_health")
if err != nil {
    log.Fatal(err)
}
healthGauge.SetTag("player_id", "player123")
scoreGauge, err := metrics.Gauge("match_score")
if err != nil {
    log.Fatal(err)
}
scoreGauge.SetTag("team", "blue")
```

### Counters
Counters represent metrics that track cumulative occurrences over time. Unlike gauges, counters only increase and are ideal for measuring
events like bytes sent, packets received, function calls, or any event that happens repeatedly. Counters accumulate values and never decrease.

**When to use:** For events that happen over time and values that only increase.

**Key characteristics:**

- Values only increase (never decrease)
- Used for calculating rates (events per second/minute)
- Reset when your application restarts
- Answer "How many?" and "How often?"

```golang
counter, err := metrics.Counter("player_deaths")
if err != nil {
    log.Fatal(err)
}
counter.Increment()        // Add 1
counter.Add(5)             // Add 5
```

**Common use cases:**
```golang
// Player activity tracking
playerJoinsCounter, err := metrics.Counter("player_joins")
if err != nil {
    log.Fatal(err)
}
playerDeathsCounter, err := metrics.Counter("player_deaths")
if err != nil {
    log.Fatal(err)
}
matchesStartedCounter, err := metrics.Counter("matches_started")
if err != nil {
    log.Fatal(err)
}

// Performance and error tracking
apiCallsCounter, err := metrics.Counter("api_calls_total")
if err != nil {
    log.Fatal(err)
}
errorCounter, err := metrics.Counter("errors_total")
if err != nil {
    log.Fatal(err)
}
errorCounter.SetTag("error_type", "network")
gcCollectionsCounter, err := metrics.Counter("garbage_collections")
if err != nil {
    log.Fatal(err)
}

// Game-specific events
itemsPickedUpCounter, err := metrics.Counter("items_picked_up")
if err != nil {
    log.Fatal(err)
}
itemsPickedUpCounter.SetTag("item_type", "weapon")
abilitiesUsedCounter, err := metrics.Counter("abilities_used")
if err != nil {
    log.Fatal(err)
}
abilitiesUsedCounter.SetTag("ability", "fireball")
```

### Timers
Timers represent duration measurements. They are ideal for tracking execution time, session duration, response times,
or any time-based metrics. Timers support derived metrics like mean, percentiles, and latest values for statistical analysis.

**When to use:** For measuring how long operations take.

**Key characteristics:**
- Measure duration of operations
- Track response times, processing duration, etc.
- Support various time units (milliseconds, seconds, duration)
- Answer "How long did that take?"

```golang
timer, err := metrics.Timer("database_query_time")
if err != nil {
    log.Fatal(err)
}

// Different ways to record time
timer.SetDuration(250 * time.Millisecond)    // Record 250 milliseconds
timer.SetMilliseconds(250.0)                 // Alternative method
timer.SetSeconds(0.25)                       // Record in seconds

// Time a function
ctx := context.Background()
err := timer.TimeFunc(ctx, func() error {
    // Your code here
    return performDatabaseQuery()
})

// Manual timing
stopTimer := timer.Start()
performOperation()
stopTimer() // Records the elapsed time
```

**Common use cases:**
```golang
// API and database performance
apiResponseTimer, err := metrics.Timer("api_response_time")
if err != nil {
    log.Fatal(err)
}
apiResponseTimer.SetTag("endpoint", "/players")
dbQueryTimer, err := factory.Timer("database_query_time")
if err != nil {
    log.Fatal(err)
}
dbQueryTimer.SetTag("operation", "select")

// Game loop and frame timing
frameTimer, err := metrics.Timer("frame_time_ms")
if err != nil {
    log.Fatal(err)
}
gameUpdateTimer, err := metrics.Timer("game_update_time")
if err != nil {
    log.Fatal(err)
}
renderTimer, err := metrics.Timer("render_time")
if err != nil {
    log.Fatal(err)
}

// Matchmaking and game operations
matchmakingTimer, err := metrics.Timer("matchmaking_duration")
if err != nil {
    log.Fatal(err)
}
gameSessionStartTimer, err := metrics.Timer("game_session_start_time")
if err != nil {
    log.Fatal(err)
}
```

### Derived Metrics

Derived metrics compute additional statistics from base metrics automatically during each capture period. They enable statistical
analysis without requiring separate metric declarations. Derived metrics are specified as optional parameters when declaring gauges
and timers (counters do not support derived metrics).

#### Latest

Tracks only the most recent values:

```golang
import (
    "log"

    sdkModule "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server"
    metricsModule "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics"
    "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/derived"
)

// Create gauge with derived metrics using the global processor
gauge, err := metrics.NewGauge("cpu_usage").
    WithMetricsProcessor(metricsModule.GetGlobalProcessor()). // Required when building directly
    WithDerivedMetrics(
        derived.NewLatest(),
    ).
    Build()
if err != nil {
    log.Fatal(err)
}
```

#### Statistical Metrics

Tracks maximum, minimum, and average values:

```golang
// Note: Derived metrics are configured during builder construction
timer, err := metrics.NewTimer("response_time").
    WithMetricsProcessor(metricsModule.GetGlobalProcessor()). // Required when building directly
    WithDerivedMetrics(
        derived.NewMax(),
        derived.NewMin(),
        derived.NewMean(),
    ).
    Build()
if err != nil {
    log.Fatal(err)
}
```

#### Percentiles

Percentile derived metrics provide predefined constants to avoid ambiguity about the scale (0-100):

```golang
// Available percentile constants (0-100 scale):
// derived.P25  = 25.0
// derived.P50  = 50.0
// derived.P75  = 75.0
// derived.P90  = 90.0
// derived.P95  = 95.0
// derived.P99  = 99.0
// derived.P999 = 99.9

// Calculate percentile values using constants
timer, err := metrics.NewTimer("response_time").
    WithMetricsProcessor(metricsModule.GetGlobalProcessor()). // Required when building directly
    WithDerivedMetrics(
        derived.NewPercentile(derived.P50, derived.P95, derived.P99),
    ).
    Build()
if err != nil {
    log.Fatal(err)
}

// You can also use custom percentile values
customTimer, err := metrics.NewTimer("custom_response_time").
    WithMetricsProcessor(metricsModule.GetGlobalProcessor()). // Required when building directly
    WithDerivedMetrics(
        derived.NewPercentile(derived.P95, 97.5, derived.P999), // Mix constants and custom values
    ).
    Build()
if err != nil {
    log.Fatal(err)
}
```

### Samplers

Samplers control which metric samples are recorded, enabling performance optimization and data volume management.

```golang
import (
    "log"

    sdkModule "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/server"
    metricsModule "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics"
    "github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/v5/metrics/samplers"
)

// Sample every value (default)
allSampler := samplers.NewAll()

// Sample 10% of values
rateSampler := samplers.NewRate(0.1)

// Apply sampling to metrics
gauge, err := metrics.NewGauge("memory_usage").
    WithMetricsProcessor(metricsModule.GetGlobalProcessor()). // Required when building directly
    WithSampler(rateSampler).
    Build()
if err != nil {
    log.Fatal(err)
}
```

### Tagging

Metrics can be tagged with key-value pairs to attach additional contextual information for filtering and grouping in
monitoring tools. Tags are applied when metrics are sent to the collector and become part of the metric's identity.

#### How Tagging Works

Tags in the Go SDK are **metadata** attached to metrics. Tags are simply key-value pairs that provide additional context for your metrics.

**Important Tag Behavior:**

- Tags can be **modified** after metric creation using `SetTag()` and `RemoveTag()`
- Only the **most recent set** of tags is attached to the metric when values are sent
- Tags are used for filtering and grouping in your monitoring system

#### Tagging Hierarchy

Tags can be set at two levels, with metric-level tags overriding global tags:

1. **Global Tags** - Applied to all metrics
2. **Metric Tags** - Applied to a specific metric instance

```golang
metrics, err := sdkModule.InitMetricsFromEnvironment()
if err != nil {
    log.Fatal(err)
}

// 1. Set global tags using the metrics instance
err = metrics.SetGlobalTag("environment", "production")
if err != nil {
    log.Printf("Failed to set global tag: %v", err)
}
err = metrics.SetGlobalTag("fleet_id", "fleet-123")
if err != nil {
    log.Printf("Failed to set global tag: %v", err)
}


// 2. Metric tags - set on individual metrics (using SetTag)
gauge, err := metrics.Gauge("player_count")
if err != nil {
    log.Fatal(err)
}
gauge.SetTag("game_mode", "ranked")
gauge.SetTag("map", "de_dust2")

// Or using individual builders for creation-time tags
gauge2, err := metrics.NewGauge("player_count").
    WithMetricsProcessor(metricsModule.GetGlobalProcessor()). // Required when building directly
    WithTag("game_mode", "ranked").        // Single key-value pair
    WithTag("map", "de_dust2").            // Another single key-value pair
    Build()
if err != nil {
    log.Fatal(err)
}

// Alternative: Using WithTags for multiple tags at once
gauge3, err := metrics.NewGauge("server_load").
    WithMetricsProcessor(metricsModule.GetGlobalProcessor()). // Required when building directly
    WithTags(map[string]string{           // Multiple key-value pairs as map
        "region":     "us-west-2",
        "game_mode":  "casual",
        "server_type": "dedicated",
    }).
    Build()
if err != nil {
    log.Fatal(err)
}

// Remove global tags
metrics.RemoveGlobalTag("environment")
```

#### Setting and Modifying Tags

```golang
// Set tags at metric creation using factory
gauge, err := metrics.Gauge("player_count")
if err != nil {
    log.Fatal(err)
}
gauge.SetTag("region", "us-west-2")
gauge.SetTag("game_mode", "ranked")

// Modify tags after creation (affects this metric instance only)
gauge.SetTag("peak_hours", "true")
gauge.RemoveTag("game_mode")

// Note: Changing tags does NOT create a new metric instance
// The same metric continues to track the same value, but subsequent values
// will be sent with the updated tags
```

#### Example: Tags Provide Context

```golang
// This creates ONE metric with tags
playerGauge, err := metrics.Gauge("player_count")
if err != nil {
    log.Fatal(err)
}
playerGauge.SetTag("server", "server-1")

// Setting different values updates the SAME metric
playerGauge.Set(10)  // player_count = 10 with tags {server: server-1}

// Changing tags affects future metric sends from the SAME metric instance
playerGauge.SetTag("server", "server-2")
playerGauge.Set(20)  // player_count = 20 with tags {server: server-2}

// There is still only ONE player_count metric instance
// The tags just changed what metadata gets sent with the metric values
```
#### Tag Validation Rules

The metrics system validates tag keys and values separately:

**Tag Keys**
- **Must start with a letter** (a-z, A-Z)
- **Allowed characters**: Letters, numbers, underscore (`_`), hyphen (`-`), period (`.`), forward slash (`/`)
- **Cannot contain colons** (`:`)
- **Maximum length**: 200 characters
- **Cannot be empty or whitespace-only**

**Tag Values**
- **Allowed characters**: Letters, numbers, underscore (`_`), hyphen (`-`), colon (`:`), period (`.`), forward slash (`/`)
- **Can contain colons** - supports tags like `env:staging:east` where value is `staging:east`
- **Maximum length**: 200 characters


## Appendix
### Choosing the Right Metric Type
The different metric types (gauge, counter, and timer) serve different purposes and have their own appropriate use cases.

| Scenario                | Metric Type   | Reason                           |
|-------------------------|---------------|----------------------------------|
| Player joins the server | Counter       | Event that accumulates over time |
| Current players online  | Gauge         | State that fluctuates up/down    |
| Time to process a match | Timer         | Duration measurement             |
| Total matches played    | Counter       | Accumulating count               |
| Server CPU usage        | Gauge         | Current resource level           |
| Database query speed    | Timer         | Performance measurement          |
| Errors encountered      | Counter       | Events to track and rate         |
| Memory consumption      | Gauge         | Current resource state           |
| Player session length   | Timer         | Duration measurement             |

**Pro tip:** Many scenarios benefit from multiple metric types:

```golang
// Player connection monitoring
connectionsCounter, err := metrics.Counter("connections_total")     // How many total
if err != nil {
    log.Fatal(err)
}

activeConnectionsGauge, err := metrics.Gauge("active_connections")  // How many now
if err != nil {
    log.Fatal(err)
}

connectionTimeTimer, err := metrics.Timer("connection_time_ms")     // How long to connect
if err != nil {
    log.Fatal(err)
}

func OnPlayerConnect() {
    start := time.Now()
    // ... connection logic ...

    connectionsCounter.Increment()                                    // Count the event
    activeConnectionsGauge.Increment()                                // Update current state
    connectionTimeTimer.SetMilliseconds(float64(time.Since(start).Milliseconds())) // Record duration
}
```