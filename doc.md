# Challenge WebSocket Manager Service - Constants & Events Documentation

## Table of Contents
1. [Challenge Status Constants](#challenge-status-constants)
2. [WebSocket Event Constants](#websocket-event-constants)
3. [System Configuration Constants](#system-configuration-constants)
4. [Question Difficulty Constants](#question-difficulty-constants)
5. [Event Types & Payloads](#event-types--payloads)
6. [WebSocket Message Types](#websocket-message-types)
7. [Error Types](#error-types)

---

## Challenge Status Constants

These constants define the various states a challenge can be in throughout its lifecycle.

```go
const (
    CHALLENGE_OPEN      = "CHALLENGEOPEN"      // Challenge is created and accepting participants
    CHALLENGE_STARTED   = "CHALLENGESTARTED"   // Challenge has begun and is active
    CHALLENGE_FORFEITED = "CHALLENGEFORFIETED" // Challenge was forfeited by participants
    CHALLENGE_ENDED     = "CHALLENGEENDED"     // Challenge completed normally
    CHALLENGE_ABANDON   = "CHALLENGEABANDON"   // Challenge was abandoned by creator
)
```

### Challenge Status Flow
```
CHALLENGE_OPEN → CHALLENGE_STARTED → CHALLENGE_ENDED
                ↓
            CHALLENGE_ABANDON
                ↓
            CHALLENGE_FORFEITED
```

---

## WebSocket Event Constants

These constants define the various WebSocket events that can be sent between client and server.

### Server Events (Outbound)
```go
const (
    PING_SERVER          = "PING_SERVER"          // Server ping response
    USER_JOINED          = "USER_JOINED"          // User joined challenge
    USER_LEFT            = "USER_LEFT"            // User left challenge
    CREATOR_ABANDON      = "CREATOR_ABANDON"      // Creator abandoned challenge
    CHALLENGE_STARTED    = "CHALLENGE_STARTED"    // Challenge has started
    OWNER_LEFT           = "OWNER_LEFT"           // Challenge owner left
    OWNER_JOINED         = "OWNER_JOINED"         // Challenge owner joined
    NEW_OWNER_ASSIGNED   = "NEW_OWNER_ASSIGNED"   // New owner was assigned
)
```

### Client Events (Inbound)
```go
const (
    JOIN_CHALLENGE    = "JOIN_CHALLENGE"    // Client wants to join challenge
    REFETCH_CHALLENGE = "REFETCH_CHALLENGE" // Client wants to refresh challenge data
)
```

---

## System Configuration Constants

These constants define system-wide configuration values and timeouts.

```go
const (
    SessionTimeout        = 30 * time.Minute  // User session timeout
    CleanupInterval       = 5 * time.Minute   // Cleanup job interval
    MaxConcurrentMatches  = 100               // Maximum concurrent challenges
    WebsocketReadTimeout  = 60 * time.Second  // WebSocket read timeout
    EmptyChallengeTimeout = 10 * time.Minute  // Timeout for empty challenges
)
```

---

## Question Difficulty Constants

These constants define the difficulty levels for challenge problems.

```go
const (
    DifficultyEasy   = "easy"     // Easy difficulty problems
    DifficultyMedium = "medium"   // Medium difficulty problems
    DifficultyHard   = "hard"     // Hard difficulty problems
)
```

---

## Event Types & Payloads

### Event Structure
```go
type Event struct {
    Type    EventType   `json:"type"`
    Payload interface{} `json:"payload"`
}
```

### Challenge Events

#### ChallengeCreatedPayload
```go
type ChallengeCreatedPayload struct {
    ChallengeID string `json:"challenge_id"`
    Title       string `json:"title"`
}
```

#### ChallengeDeletedPayload
```go
type ChallengeDeletedPayload struct {
    ChallengeID string `json:"challenge_id"`
}
```

#### ChallengeStatusChangedPayload
```go
type ChallengeStatusChangedPayload struct {
    ChallengeID string          `json:"challenge_id"`
    Status      ChallengeStatus `json:"status"`
}
```

### User Events

#### UserJoinedPayload
```go
type UserJoinedPayload struct {
    UserID string `json:"user_id"`
}
```

#### UserLeftPayload
```go
type UserLeftPayload struct {
    UserID string `json:"user_id"`
}
```

#### UserForfeitedPayload
```go
type UserForfeitedPayload struct {
    UserID string `json:"user_id"`
}
```

### Submission Events

#### ProblemSubmittedPayload
```go
type ProblemSubmittedPayload struct {
    UserID    string `json:"user_id"`
    ProblemID string `json:"problem_id"`
    Score     int    `json:"score"`
}
```

#### LeaderboardUpdatedPayload
```go
type LeaderboardUpdatedPayload struct {
    Leaderboard []*LeaderboardEntry `json:"leaderboard"`
}
```

### System Events

#### TimeUpdatePayload
```go
type TimeUpdatePayload struct {
    RemainingTime int64 `json:"remaining_time"` // In seconds
}
```

#### ErrorPayload
```go
type ErrorPayload struct {
    Message string `json:"message"`
}
```

---

## WebSocket Message Types

### Inbound Messages

#### JoinChallengePayload
```go
type JoinChallengePayload struct {
    UserId      string `json:"userId"`
    Type        string `json:"type"`
    ChallengeId string `json:"challengeId"`
    Password    string `json:"password"`
    Token       string `json:"token"`
}
```

#### RefetchChallengePayload
```go
type RefetchChallengePayload struct {
    UserId      string `json:"userId"`
    Type        string `json:"type"`
    ChallengeId string `json:"challengeId"`
}
```

### Outbound Messages

#### GenericResponse
```go
type GenericResponse struct {
    Success bool           `json:"success"`
    Status  int            `json:"status"`
    Payload map[string]any `json:"payload"`
    Error   *ErrorInfo     `json:"error"`
}
```

#### WsMessage
```go
type WsMessage struct {
    Type    string         `json:"type"`
    Payload map[string]any `json:"payload"`
}
```

---

## Error Types

### ErrorInfo Structure
```go
type ErrorInfo struct {
    ErrorType string `json:"errorType"`  // Error category
    Code      int    `json:"code"`       // HTTP-like error code
    Message   string `json:"message"`    // Human-readable message
    Details   string `json:"details"`    // Additional error details
}
```

### Common Error Types
- `CHALLENGENOTFOUND` - Challenge does not exist
- `NOTCREATOR` - User is not the challenge creator
- `CHALLENGEABANDONFAILED` - Failed to abandon challenge
- `INVALIDTOKEN` - Authentication token is invalid
- `CHALLENGEFULL` - Challenge has reached maximum participants
- `WRONGPASSWORD` - Incorrect challenge password

---

## Event Flow Examples

### User Joining Challenge
1. Client sends `JOIN_CHALLENGE` with `JoinChallengePayload`
2. Server validates request
3. Server broadcasts `USER_JOINED` or `OWNER_JOINED` to all participants
4. Server responds with `GenericResponse`

### Challenge Abandonment
1. Creator calls abandon challenge API
2. Server updates challenge status to `CHALLENGE_ABANDON`
3. Server broadcasts `CREATOR_ABANDON` to all participants
4. Server persists challenge to MongoDB
5. Server cleans up Redis data

### Challenge Lifecycle
1. Challenge created with status `CHALLENGE_OPEN`
2. Users join via `JOIN_CHALLENGE` events
3. Creator starts challenge → status becomes `CHALLENGE_STARTED`
4. Challenge ends → status becomes `CHALLENGE_ENDED`
5. Challenge data persisted to MongoDB
6. Redis data cleaned up

---

## WebSocket Connection States

### Connection Context
```go
type WsContext struct {
    Conn    *websocket.Conn  // WebSocket connection
    Payload map[string]any   // Message payload
    UserID  string           // Authenticated user ID
    State   *State           // Application state
}
```

### Application State
```go
type State struct {
    Redis      *repo.RedisRepository    // Redis repository for real-time data
    Repo       *repo.MongoRepository    // MongoDB repository for persistence
    LocalState *state.LocalStateManager // Local state for non-serializable data
}
```

---

## Notes

- All timestamps are Unix timestamps (seconds since epoch)
- WebSocket messages use JSON format
- Redis is used for real-time challenge data
- MongoDB is used for historical challenge persistence
- Local state manages WebSocket connections and non-serializable data
- Challenge passwords are auto-generated for private challenges
- Session management includes automatic cleanup of inactive sessions