[![Review Assignment Due Date](https://classroom.github.com/assets/deadline-readme-button-22041afd0340ce965d47ae6ef1cefeee28c7c493a6346c4f15d667ab976d596c.svg)](https://classroom.github.com/a/YLXuze2Y)


# 🏋️‍♂️ Forum FitNation

Bienvenue sur **FitNation**, la plateforme communautaire dédiée aux passionnés de fitness, de musculation et de bien-être ! 

L'objectif de ce forum est de permettre aux sportifs de tous niveaux de partager leurs entraînements, de poser des questions, d'échanger des conseils et de suivre leur progression avec la communauté.

## ✨ Fonctionnalités principales

- 🔐 **Authentification complète** : Inscription, connexion sécurisée (hachage PBKDF2), et système de réinitialisation de mot de passe par email.
- 👤 **Profils personnalisés** : Chaque membre possède un profil avec une bio, un avatar personnalisé, et un suivi de ses statistiques (nombre de posts, commentaires et likes reçus).
- 📝 **Publications & Recherche** : Création de posts avec des tags, recherche par mots-clés, et tri dynamique (par date, popularité).
- 💬 **Interactions** : Les membres peuvent commenter les posts et ajouter des "likes" de manière fluide.
- 🌐 **Réseau** : Un annuaire permet de découvrir les autres membres de la communauté et de consulter leurs publications.
- 🛡️ **Gestion des erreurs** : Pages personnalisées (404, 400, 500) pour garantir une expérience utilisateur agréable même quand les choses se passent mal.

## 🛠️ Stack Technique

- **Backend** : Go (Golang) avec `net/http` et `html/template`
- **Base de données** : SQLite (fichier `.db` local)
- **Frontend** : HTML5, CSS3, Vanilla JS
- **Architecture** : Modèle MVC simplifié (Routes, Modèles, Vues)

## 🚀 Installation & Lancement

### Prérequis
- [Go](https://golang.org/dl/) (version 1.20 ou supérieure recommandée)

### 1. Cloner le projet
```bash
git clone https://github.com/Lyon-Ynov-Campus/forum-fitnation.git
cd forum-fitnation
```

### 2. Configuration de l'environnement
Créez un fichier `.env` à la racine du projet en vous basant sur ce modèle :

```env
# Base de données
FITNATION_DB=fitnation.db

# Configuration SMTP (pour l'envoi d'emails de réinitialisation de mot de passe)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=votre_email@example.com
SMTP_PASSWORD=votre_mot_de_passe
SMTP_FROM=noreply@fitnation.com
```

### 3. Lancer le serveur
Démarrez l'application avec la commande suivante :

```bash
go run ./cmd/server/
```

Le serveur sera alors accessible à l'adresse : **[http://localhost:8000](http://localhost:8000)**

## 📂 Structure du projet

- `/cmd/server/` : Point d'entrée de l'application (routes, configuration serveur).
- `/internal/` : Cœur de l'application (modèles de données, interactions avec la base de données).
- `/web/static/` : Fichiers statiques (CSS, images, et uploads d'avatars).
- `/web/templates/` : Fichiers HTML (Vues).

## 👨‍💻 Contribution
Ce projet est réalisé dans le cadre du cursus de Lyon Ynov Campus. 
N'hésitez pas à ouvrir des *Issues* ou des *Pull Requests* si vous souhaitez proposer des améliorations !
