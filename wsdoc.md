## WebSocket Event Reference (BFF style)

### Envelope
- Requests: `{ type: string, payload: object }`
- Responses: `{ type: string, status: "ok" | "error", payload?: object, error?: { code?: string, message: string } }`

---

### 1) PING_SERVER
- Request: `{ "type": "PING_SERVER", "payload": {} }`
- Response: `{ "type": "PING_SERVER", "status": "ok", "payload": { "message": "pong" } }`

---

### 2) JOIN_CHALLENGE (no auth token required; returns challengeToken)
- Request: `{ userId: string, challengeId: string, password?: string, token: string }`
- Response: `{ "type": "JOIN_CHALLENGE", "status": "success", "payload": { userId, challengeId, challenge, challengeToken } }`
- Example:
  - Req: `{ "type": "JOIN_CHALLENGE", "payload": { "userId": "u1", "challengeId": "c1", "password": "abc", "token": "Bearer ..." } }`
  - Res: `{ "type": "JOIN_CHALLENGE", "status": "success", "payload": { "userId": "u1", "challengeId": "c1", "challenge": { "challengeId": "c1", "title": "30-min sprint" }, "challengeToken": "eyJ..." } }`

---

### 3) RETRIEVE_CHALLENGE (auth required)
- Request: `{ userId: string, challengeId: string, challengeToken: string }`
- Response: `{ "type": "RETRIEVE_CHALLENGE", "status": "ok", "payload": { userId, challengeId, challenge } }`

---

### 4) WHOLECHAT (auth required)
- Request: `{ userId: string, challengeId: string, challengeToken: string }`
- Response: `{ "type": "WHOLECHAT", "status": "ok", "payload": { chat: ChatMessage[] } }`
- Example:
  - Res: `{ "type": "WHOLECHAT", "status": "ok", "payload": { "chat": [ { "userId": "u2", "profilePic": "https://.../u2.png", "message": "gl hf", "time": 1730829600 } ] } }`

---

### 5) WHOLENOTIFICATION (auth required)
- Request: `{ userId: string, challengeId: string, challengeToken: string }`
- Response: `{ "type": "WHOLENOTIFICATION", "status": "ok", "payload": { notifications: Notification[] } }`
- Example:
  - Res: `{ "type": "WHOLENOTIFICATION", "status": "ok", "payload": { "notifications": [ { "type": "NEW_SUBMISSION", "message": "u2 scored 10 on p123", "time": 1730829600 } ] } }`

---

### 6) WHOLELEADERBOARD (auth required)
- Request: `{ userId?: string, challengeId: string, limit?: number, challengeToken: string }`
- Response: `{ "type": "WHOLELEADERBOARD", "status": "ok", "payload": { challengeId, leaderboard: LeaderboardEntry[] } }`

---

### 7) GET_CHALLENGE_DATA (auth required)
- Request: `{ userId: string, challengeId: string, challengeToken: string }`
- Response: `{ "type": "GET_CHALLENGE_DATA", "status": "ok", "payload": { challenge, leaderboard } }`
- Example:
  - Req: `{ "type": "GET_CHALLENGE_DATA", "payload": { "userId": "u1", "challengeId": "c1", "challengeToken": "eyJ..." } }`
  - Res: `{ "type": "GET_CHALLENGE_DATA", "status": "ok", "payload": { "challenge": { "challengeId": "c1" }, "leaderboard": [ { "userId": "u1", "totalScore": 42, "rank": 1, "problemsCompleted": 3 } ] } }`

---

### 8) GET_CHALLENGE_MIN (auth required)
- Request: `{ userId: string, challengeId: string, challengeToken: string }`
- Response: `{ "type": "GET_CHALLENGE_MIN", "status": "ok", "payload": { challengeId, title, status, isPrivate, createdAt, startTime, timeLimit } }`

---

### 9) GET_PARTICIPANT_DATA (auth required)
- Request: `{ userId: string, challengeId: string, challengeToken: string }`
- Response: `{ "type": "GET_PARTICIPANT_DATA", "status": "ok", "payload": { metadata, submissions?: { [problemId]: { submissionId, points, timeTakenMillis } } } }`
- Example:
  - Req: `{ "type": "GET_PARTICIPANT_DATA", "payload": { "userId": "u1", "challengeId": "c1", "challengeToken": "eyJ..." } }`
  - Res: `{ "type": "GET_PARTICIPANT_DATA", "status": "ok", "payload": { "metadata": { "totalScore": 42, "problemsAttempted": 3 }, "submissions": { "p123": { "submissionId": "s1", "points": 20, "timeTakenMillis": 90000 } } } }`

---

### 10) GET_PARTICIPANTS_DATA (auth required)
- Request: `{ userId: string, challengeId: string, challengeToken: string }`
- Response: `{ "type": "GET_PARTICIPANTS_DATA", "status": "ok", "payload": { participants: { [userId]: ParticipantMetadata } } }`

---

### 11) CURRENT_LEADERBOARD (auth required)
- Request: `{ userId?: string, challengeId: string, limit?: number, challengeToken: string }`
- Response: `{ "type": "CURRENT_LEADERBOARD", "status": "ok", "payload": { challengeId, leaderboard: LeaderboardEntry[] } }`

---

### 12) PUSHNEWCHAT (auth required)
- Request: `{ userId: string, challengeId: string, profilePic: string, message: string, challengeToken: string }`
- Response: `{ "type": "PUSHNEWCHAT", "status": "ok", "payload": { "message": "sent" } }`
- Example:
  - Req: `{ "type": "PUSHNEWCHAT", "payload": { "userId": "u1", "challengeId": "c1", "profilePic": "https://.../u1.png", "message": "hello!", "challengeToken": "eyJ..." } }`
  - Res: `{ "type": "PUSHNEWCHAT", "status": "ok", "payload": { "message": "sent" } }`
- Broadcast: `{ "type": "PUSHNEWCHAT", "success": true, "payload": { "userId": "u1", "profilePic": "https://.../u1.png", "message": "hello!", "time": 1730829600 } }`

---

### 13) PUSHNEWNOTIFICATION (auth required)
- Request: `{ challengeId: string, type: string, message: string, time: number, challengeToken: string }`
- Response: `{ "type": "PUSHNEWNOTIFICATION", "status": "ok", "payload": { "message": "accepted" } }`
- Example:
  - Req: `{ "type": "PUSHNEWNOTIFICATION", "payload": { "challengeId": "c1", "type": "GAME_FINISHED", "message": "Challenge ended!", "time": 1730829600, "challengeToken": "eyJ..." } }`
  - Res: `{ "type": "PUSHNEWNOTIFICATION", "status": "ok", "payload": { "message": "accepted" } }`
- Broadcast: `{ "type": "PUSHNEWNOTIFICATION", "success": true, "payload": { "type": "GAME_FINISHED", "message": "Challenge ended!", "time": 1730829600 } }`

---

## Broadcast Events

- `PUSH_SUBMISSION`: `{ "type": "PUSHSUBMISSION", "success": true, "payload": { challengeId, userId, problemId, score, newRank, time } }`
- `LEADERBOARD_UPDATE`: `{ "type": "LEADERBOARD_UPDATE", "success": true, "payload": { challengeId, leaderboard, updatedUser, time } }`
- `GAME_FINISHED`: `{ "type": "GAME_FINISHED", "success": true, "payload": { challengeId, time } }`
- `USER_JOINED`: `{ "type": "USER_JOINED", "success": true, "payload": { userId, challengeId, time } }`
- `USER_LEFT`: `{ "type": "USER_LEFT", "success": true, "payload": { userId, challengeId, time } }`
- `CREATOR_ABANDON`: `{ "type": "CREATOR_ABANDON", "success": true, "payload": { challengeId, userId, time } }`
- `CHAT_MESSAGE`: `{ "type": "CHAT_MESSAGE", "success": true, "payload": { challengeId, message: { userId, profilePic, message, time } } }`

---

## Type Hints
- `ChallengeDocument`: `{ challengeId, creatorId, createdAt, title, isPrivate, status, timeLimit, startTime, participants, submissions, leaderboard, notifications, chat }`
- `ParticipantMetadata`: `{ totalScore, problemsAttempted, joinTime, lastConnected, problemsDone? }`
- `Submission`: `{ submissionId, timeTaken: number(ms), points }`
- `LeaderboardEntry`: `{ userId, totalScore, problemsCompleted, rank }`
- `Notification`: `{ type, message, time }`
- `ChatMessage`: `{ userId, profilePic, message, time }`

---

## Error Format
- `{ "type": "<EVENT>", "status": "error", "error": { "code": "SOME_CODE", "message": "Readable message" } }`


