[![Review Assignment Due Date](https://classroom.github.com/assets/deadline-readme-button-22041afd0340ce965d47ae6ef1cefeee28c7c493a6346c4f15d667ab976d596c.svg)](https://classroom.github.com/a/YLXuze2Y)

# Forum FitNation

Plateforme communautaire dédiée aux passionnés de fitness. Partagez vos entraînements, posez vos questions, échangez des conseils.

**Démo live : [https://forum-fitnation.onrender.com](https://forum-fitnation.onrender.com)**

> Instance gratuite Render — premier chargement peut prendre ~30 secondes (spin-down après inactivité).

## Fonctionnalités

### Utilisateurs
- Inscription avec confirmation de mot de passe (conformité CNIL : 8 caractères min, majuscule, minuscule, chiffre)
- Connexion par email ou pseudo
- Réinitialisation de mot de passe par email (lien expirant 1h)
- Profil personnalisable : pseudo, email, mot de passe, avatar
- Visualisation de ses posts et commentaires depuis le profil
- Suppression de compte (cascade sur toutes les données)
- Navigation en lecture seule sans connexion

### Posts
- Création, modification, suppression de posts (titre, contenu, tags, image)
- Page dédiée par post avec commentaires
- Likes sans rechargement de page (API fetch)
- Modification et suppression des commentaires

### Recherche & Filtres
- Recherche de posts par titre (server-side)
- Tri par date, popularité (likes) ou activité (commentaires)
- Filtres : période (aujourd'hui / semaine / mois), likes minimum, commentaires minimum

### Réseau
- Annuaire des membres avec stats (posts, commentaires, likes reçus)
- Barre de recherche de membres

### Admin
- Accès restreint via `/admin/login` (identifiants configurables par variables d'environnement)
- Bannissement / débannissement de comptes (session invalidée immédiatement)
- Suppression de comptes et de posts
- Lien Admin visible uniquement pour les comptes `@ynov.com`

## Stack technique

| Couche | Technologie |
|--------|------------|
| Backend | Go 1.24 — `net/http`, `html/template` |
| Base de données | SQLite (`modernc.org/sqlite`) |
| Frontend | HTML5, CSS3, Vanilla JS |
| Déploiement | Render (Docker runtime) |

## Sécurité

- **CSRF** : protection HMAC sur toutes les mutations (formulaires + API fetch via `X-CSRF-Token`)
- **Mots de passe** : PBKDF2-SHA256, 120 000 itérations, sel aléatoire 16 octets
- **Sessions** : cookies `HttpOnly`, `SameSite=Lax`, `Secure` automatique en HTTPS
- **Avatars** : validation par magic bytes (JPEG/PNG/GIF/WebP), limite 2 Mo
- **En-têtes** : `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, HSTS en HTTPS
- **Admin** : tokens en mémoire protégés par `sync.RWMutex`, `ADMIN_USERNAME`/`ADMIN_PASSWORD` via variables d'environnement

## Installation locale

**Prérequis** : Go 1.20+

```bash
git clone https://github.com/Lyon-Ynov-Campus/forum-fitnation.git
cd forum-fitnation
```

Créer un fichier `.env` à la racine :

```env
FITNATION_DB=fitnation.db
PORT=8000
CSRF_SECRET=une_chaine_aleatoire_longue

# SMTP (optionnel — sans config, le lien reset s'affiche dans les logs)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=votre_email@example.com
SMTP_PASSWORD=votre_mot_de_passe
SMTP_FROM=noreply@fitnation.com

# Admin (défaut si non renseigné : admin / admin1234)
ADMIN_USERNAME=admin
ADMIN_PASSWORD=motdepasse_securise
```

Lancer le serveur :

```bash
go run ./cmd/server/
```

Accès : [http://localhost:8000](http://localhost:8000)

## Déploiement (Render)

Le projet inclut un `Dockerfile` et un `render.yaml`. Pour déployer :

1. Créer un **Web Service** sur [render.com](https://render.com) depuis ce repo
2. **Build Command** : `go build -tags netgo -ldflags '-s -w' -o app ./cmd/server`
3. **Start Command** : `./app`
4. Définir les variables d'environnement (`CSRF_SECRET`, `ADMIN_USERNAME`, `ADMIN_PASSWORD`, SMTP…)

## Structure du projet

```
forum-fitnation/
├── cmd/server/
│   ├── main.go        # Routes, handlers principaux, sécurité
│   └── admin.go       # Handlers admin (login, ban, suppression)
├── internal/
│   ├── database/      # Couche SQLite
│   └── models/        # Structs de données
├── web/
│   ├── static/        # CSS, JS, images
│   └── templates/     # Vues HTML
├── Dockerfile
└── render.yaml
```

## Projet

Réalisé dans le cadre du cursus Lyon Ynov Campus.
