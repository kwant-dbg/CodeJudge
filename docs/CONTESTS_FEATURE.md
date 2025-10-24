# 🏆 CodeJudge Contests Feature

A complete competitive programming contest system inspired by Codeforces, integrated into CodeJudge.

## ✨ Features

### Core Functionality
- ✅ **Contest Management** - Create, view, and manage programming contests
- ✅ **Time-based States** - Automatic contest status (upcoming/active/finished)
- ✅ **Real-time Leaderboard** - Live rankings with automatic updates during active contests
- ✅ **Leaderboard Freeze** - Freeze leaderboard in final minutes (configurable)
- ✅ **User Registration** - Participants can register for contests
- ✅ **Custom Scoring** - Assign different point values to each problem
- ✅ **Contest Problems** - Multiple problems per contest
- ✅ **Penalty System** - Time-based penalties in scoring
- ✅ **Admin Controls** - Admin-only contest creation and management

### User Interface
- 📱 **Responsive Design** - Works on desktop, tablet, and mobile
- 🎨 **Beautiful UI** - Modern, clean interface with smooth animations
- 🏅 **Status Badges** - Visual indicators for contest states
- 📊 **Leaderboard Table** - Clean, sortable rankings
- 🎯 **Contest Cards** - Easy-to-read contest information
- ⚡ **Auto-refresh** - Leaderboard updates every 30 seconds during active contests

## 🗄️ Database Schema

### Tables Created

```sql
-- Contests
CREATE TABLE contests (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    created_by INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    freeze_time INTEGER DEFAULT 0
);

-- Contest Problems (many-to-many)
CREATE TABLE contest_problems (
    id SERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    points INTEGER NOT NULL DEFAULT 100,
    order_num INTEGER NOT NULL,
    UNIQUE(contest_id, problem_id)
);

-- Contest Participants
CREATE TABLE contest_participants (
    contest_id INTEGER NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (contest_id, user_id)
);

-- Updated Submissions Table
ALTER TABLE submissions 
ADD COLUMN contest_id INTEGER REFERENCES contests(id) ON DELETE SET NULL,
ADD COLUMN user_id INTEGER,
ADD COLUMN points INTEGER DEFAULT 0;
```

## 🔌 API Endpoints

### Public Endpoints (No Auth Required)

```
GET /api/contests
- List all contests
- Query params: ?status={upcoming|active|finished}
- Response: Array of contest objects

GET /api/contests/{id}
- Get contest details
- Response: Contest object with metadata

GET /api/contests/{id}/problems
- Get problems in a contest
- Response: Array of problems with points

GET /api/contests/{id}/leaderboard
- Get contest leaderboard
- Response: Ranked list of participants with scores
- Note: Automatically handles leaderboard freeze
```

### Protected Endpoints (Requires Authentication)

```
POST /api/contests/{id}/register
- Register for a contest
- Headers: Authorization: Bearer <token>
- Response: Success message

POST /api/submissions
- Submit solution (now supports contest_id)
- Body: { problem_id, source_code, language, contest_id? }
- Automatic score calculation for contest submissions
```

### Admin Endpoints (Requires Admin Role)

```
POST /api/contests
- Create new contest
- Body: { title, description, start_time, end_time, freeze_time? }
- Response: Created contest object

POST /api/contests/{id}/problems
- Add problem to contest
- Body: { problem_id, points, order_num }
- Response: Contest problem mapping
```

## 🎮 User Guide

### For Participants

1. **Browse Contests**
   - Visit `/contests.html`
   - Filter by status: All, Upcoming, Active, Finished
   - Click any contest card to view details

2. **Register for Contest**
   - Open contest details page
   - Click "Register" button (requires login)
   - You're now registered!

3. **Solve Problems**
   - View contest problems and their point values
   - Click any problem to open it
   - You'll see a contest banner at the top
   - Submit your solution as usual
   - Points are automatically calculated and added to leaderboard

4. **Track Your Rank**
   - View leaderboard in real-time
   - See your rank, score, problems solved, and penalty time
   - Top 3 positions are highlighted (🥇🥈🥉)

### For Admins

1. **Create Contest**
   - Login as admin
   - Go to `/contests.html`
   - Click "Create Contest" button
   - Fill in contest details:
     - Title (required)
     - Description
     - Start time (required)
     - End time (required)
     - Leaderboard freeze time (optional, in minutes)

2. **Add Problems**
   - On contest creation form, add problems
   - Select problem from dropdown
   - Assign point value (default: 100)
   - Set problem order
   - Can add multiple problems

3. **Launch Contest**
   - Submit the form
   - Contest is created immediately
   - Status automatically changes based on time
   - Participants can register until contest ends

## 📊 Scoring System

### How Scores are Calculated

1. **Base Points**: Each problem has assigned points (set by admin)
2. **Acceptance**: Full points awarded only on "Accepted" verdict
3. **Penalty**: Time from contest start to successful submission (in minutes)
4. **Ranking**: Sorted by:
   - Total score (DESC)
   - Problems solved (DESC)
   - Earliest submission time (ASC)

### Leaderboard Freeze

- Set freeze time when creating contest (e.g., 60 minutes)
- During last X minutes, leaderboard shows standings from freeze point
- Adds excitement and prevents last-minute sniping
- Full results revealed after contest ends

## 🚀 Quick Start

### 1. Start the System
```powershell
docker-compose up -d --build
```

### 2. Create Admin Account
```powershell
# Access the app
http://localhost:8080

# Register as first user
# Use the admin panel to make yourself admin
```

### 3. Create Your First Contest
```
1. Login as admin
2. Go to Contests page (🏆 button)
3. Click "Create Contest"
4. Fill in details
5. Add 3-5 problems with different point values
6. Submit
```

### 4. Test as Participant
```
1. Open contest in incognito/another browser
2. Register for contest
3. Solve a problem
4. Check leaderboard
```

## 🎯 Best Practices

### Contest Duration
- **Short contests**: 1-2 hours (3-4 problems)
- **Medium contests**: 3-4 hours (5-7 problems)
- **Long contests**: 24-48 hours (8-10 problems)

### Problem Points
- Easy: 100-200 points
- Medium: 300-500 points
- Hard: 600-1000 points

### Freeze Time
- Short contests: 15-30 minutes
- Long contests: 60-120 minutes
- Set to 0 for no freeze

## 🔧 Technical Details

### Handler: `handlers/contests.go`
- `ContestsHandler` struct with database manager
- Methods for all contest operations
- Automatic status calculation
- Optimized leaderboard queries with JOINs

### Frontend Files
- `contests.html` - Main contest listing page
- `contest-detail.html` - Contest view with leaderboard
- `create-contest.html` - Admin contest creation form
- `problem.html` - Updated to support contest submissions

### Key Features in Code
- **SQL Transactions**: Safe contest creation
- **Prepared Statements**: Optimized queries
- **Real-time Updates**: Auto-refresh leaderboard
- **Time-based Logic**: Automatic contest state management
- **Security**: JWT auth, admin-only endpoints

## 🐛 Troubleshooting

### Leaderboard not updating
- Check if contest is actually active
- Verify submissions have contest_id set
- Check browser console for errors

### Can't create contest
- Ensure you're logged in as admin
- Check start_time < end_time
- Verify database connection

### Submissions not counting for contest
- Ensure you're clicking problem from contest page
- Check URL has `?contest_id=X` parameter
- Verify contest is active (not finished)

## 📝 Example Contest

```json
{
  "title": "Beta Round #5",
  "description": "Solve algorithmic problems in 2 hours!",
  "start_time": "2025-10-25T14:00:00Z",
  "end_time": "2025-10-25T16:00:00Z",
  "freeze_time": 30,
  "problems": [
    { "problem_id": 1, "points": 100, "order": 1 },
    { "problem_id": 2, "points": 200, "order": 2 },
    { "problem_id": 3, "points": 300, "order": 3 }
  ]
}
```

## 🎉 Enjoy Your Contest Platform!

Your CodeJudge instance now has a full-featured contest system. Happy coding! 🚀
