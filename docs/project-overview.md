## Project Overview: LifeOS

LifeOS is an always-on, second-monitor dashboard designed to engineer discipline into a remote software developer's daily routine. It bridges the gap between physical transformation and technical skill acquisition by treating personal habits as trackable, scalable systems.

Built specifically to solve the friction of a disorganized lifestyle, LifeOS eliminates the need for multiple tracking apps, notebooks, and timers. It provides a single, unified interface that resets every morning, guiding you through home workouts, deep learning blocks, and your professional remote work hours.

---

### Core Features & Functionality

#### 1. The Command Center (Dynamic Daily Timeline)

This is the heart of LifeOS. It visualizes your day as a structured timeline, highlighting exactly what you should be doing at any given minute.

* **Context-Aware Scheduling:** It automatically adjusts based on your Sunday–Thursday workweek (9:30 AM - 6:00 PM).
* **Active Block Highlighting:** Whether it's your 6:15 AM workout, your 8:00 AM Go/AI learning block, or a professional meeting, the dashboard keeps the current task front and center.
* **The Daily Reset Engine:** Utilizing a "lazy evaluation" architecture on the Go backend, LifeOS automatically archives yesterday's data and generates a clean, fresh log the moment you open the dashboard each morning.

#### 2. Deep Work & Pomodoro Engine

To level up in DevOps, Go, and AI, you need unbroken focus. This feature handles the cognitive load of time management.

* **Real-Time WebSocket Sync:** The countdown timer is driven by Go channels on the backend, ensuring perfect synchronization even if your browser tab goes to sleep.
* **System Integration:** When a 90-minute focus block ends, LifeOS triggers native browser notifications and plays a subtle audio chime, ensuring you take necessary screen breaks.

#### 3. Home Gym & Telemetry Log

You don't need a gym; you just need to track the equipment you have. This module replaces fitness apps with a custom interface built for your specific hardware.

* **Equipment-Specific Routines:** Workouts are pre-configured for your 3.1 KG barbell, 1 KG dumbbells, weight plates (up to 18.1 KG total load), and the color-coded Pushup Board (targeting chest and triceps).
* **Guided Player Mode:** When you start a workout, the UI expands to show the current exercise, a rest timer, and embedded looping video tutorials for proper form.
* **Frictionless Logging:** Input your reps and weight directly into the dashboard. The data saves instantly to your PostgreSQL database.

#### 4. Habit, Nutrition & Hydration Tracking

Designed to fix the "lazy and messy" approach to basic health inputs.

* **One-Click Hydration:** A dedicated widget on the dashboard to log water intake in increments (e.g., +250ml) to ensure you hit your daily hydration goals.
* **Binary Habit Toggles:** Simple checkboxes for daily non-negotiables (e.g., "8 Hours Sleep," "Hit Protein Goal").
* **Caloric Deficit Tracking:** A streamlined interface to track your daily weight and ensure you are trending toward your 67 KG target.

#### 5. Analytics & Progress Visualization

Data without visualization is useless. This module aggregates your daily logs into actionable feedback.

* **Weight & Physique Trajectory:** A line chart plotting your progress from 72 KG down to 67 KG over time.
* **Consistency Heatmaps:** A GitHub-style contribution graph showing your habit streaks (green squares for perfect days, grey for missed days).
* **Volume Tracking:** Bar charts displaying your weekly workout volume to ensure you are progressively overloading your muscles for visible growth in your chest, shoulders, and arms.

---

### Technical Foundation

LifeOS is built on an enterprise-grade stack designed for ultra-low resource consumption and high performance, directly aligning with your goal to master modern backend architectures:

* **Frontend:** Vite + React (TypeScript), Tailwind CSS, and Shadcn UI for a sleek, dark-mode optimized interface.
* **Backend:** Go (Fiber) for lightning-fast API responses and minimal memory usage while idling on your second monitor.
* **Database:** PostgreSQL paired with GORM (Go's native ORM) for strict, code-first relational data management.
* **Security:** Native Go authentication using `bcrypt` and HTTP-only JWT cookies—zero reliance on third-party services like Firebase.
