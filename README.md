# Go Shop

A full-stack online shop built from scratch in Go with a SQLite backend — catalog, cart, checkout, product reviews, and an admin panel. Built as a hands-on project to learn backend development and system design.

**Live demo:** https://shop3-production.up.railway.app

## Features

- **Product catalog** with case-insensitive search (works with Cyrillic too), price sorting, and an in-stock filter, all combinable via URL parameters
- **Shopping cart** with live quantity updates and instant total recalculation (JavaScript + background save), plus stock-limit protection
- **Checkout & orders** — placing an order records it and decrements stock; users see their full order history
- **Product reviews** — only customers who actually bought a product can review it (verified with a SQL JOIN across orders), one review per person, with an average rating on each product page
- **Authentication** — registration and login with salted, hashed passwords; sessions stored in the database so they survive restarts
- **Admin panel** — a role-protected area to add products (with image upload from your computer), edit prices and stock in bulk, and review all data
- **In-process catalog cache** — a cache-aside layer with TTL and cache invalidation on product changes

## Tech stack

- **Language:** Go (standard library net/http, no web framework)
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGO)
- **Templates:** Go's built-in html/template
- **Frontend:** server-rendered HTML with vanilla CSS and a touch of JavaScript
- **Hosting:** Railway (with a persistent volume for the database and uploaded images)

## Running locally

You will need Go 1.22 or newer.

    git clone https://github.com/yulaman-code/shop.git
    cd shop
    go run .

Then open http://localhost:8080 in your browser. The database and a few sample products are created automatically on first run.

### Configuration

Optional environment variables:

- PORT — port to listen on (default 8080)
- DB_PATH — path to the SQLite database file (default shop.db)
- UPLOAD_DIR — directory for uploaded product images (default static/images)
- ADMIN_USERNAME — username to grant admin rights on startup

To make yourself an admin locally, register a user, then restart with:

    ADMIN_USERNAME=yourname go run .

## Project structure

    main.go        entry point, routes
    handlers.go    HTTP handlers (catalog, cart, orders, product page)
    store.go       database setup and product/order queries
    sessions.go    auth middleware and session handling
    reviews.go     product reviews (queries + rating logic)
    admin.go       admin actions (create/update products)
    upload.go      image upload handling
    cache.go       in-process catalog cache
    templates/     HTML templates
    static/        CSS and images

## Notes

This is a learning project, so some choices favour clarity over production-readiness. For example, search and sorting are done in Go rather than SQL (fine for a small catalog, but a real store would push that work into the database). The caching layer is intentionally simple to demonstrate the cache-aside pattern.
