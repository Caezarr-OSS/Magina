# Magina

Magina est un outil en ligne de commande (CLI) pour la gestion et la migration d'images OCI (Open Container Initiative) entre différents registries. Il utilise un format de configuration personnalisé BRMS (Block Relation Mapping Syntax) pour définir les mappings d'images.

## Fonctionnalités

- Export d'images depuis un registry source
- Import d'images vers un registry destination
- Transfert complet d'images entre registries
- Support de l'authentification registry
- Validation de configuration BRMS
- Nettoyage automatique en cas d'erreur
- Support de la reprise sur erreur
- Logging détaillé et configurable
- Fonctionne sans runtime de conteneur (Docker ou Podman non requis)

## Prérequis

### Système
- Windows, Linux ou macOS (amd64 ou arm64)
- 1GB d'espace disque minimum
- 512MB RAM minimum (1GB recommandé)

### Réseau
- Accès Internet pour les registries distants
- Ports 80/443 accessibles
- Configuration proxy si nécessaire

### Note sur les Runtimes de Conteneurs
Magina fonctionne de manière autonome et ne nécessite pas de runtime de conteneur (Docker, Podman, etc.) car il interagit directement avec les registries via le protocole OCI standard. Cependant, si vous souhaitez utiliser les images localement après les avoir téléchargées, vous aurez besoin d'un runtime compatible OCI comme Docker ou Podman.

## Installation

### Depuis les releases
```bash
# Télécharger la dernière version depuis GitHub Releases
# Remplacer <platform> par linux, darwin ou windows
# Remplacer <arch> par amd64 ou arm64
curl -LO https://github.com/Caezarr-OSS/magina/releases/latest/download/magina-<platform>-<arch>.tar.gz
tar xzf magina-<platform>-<arch>.tar.gz
chmod +x magina-<platform>-<arch>
mv magina-<platform>-<arch> /usr/local/bin/magina  # ou autre répertoire dans le PATH
```

### Depuis les sources
```bash
# Cloner le repository
git clone https://github.com/Caezarr-OSS/magina.git
cd magina

# Installer Task
sh -c "$(curl --location https://taskfile.dev/install.sh)"

# Compiler pour votre plateforme
task build

# Ou compiler pour toutes les plateformes
task release
```

## Utilisation

### Format BRMS
```brms
# Format général
[protocol://source-registry|protocol://dest-registry]
image1:tag1|newimage1:tag1
image2:tag2|newimage2:tag2

# Exemple concret
[https://registry.company.com|https://docker.io]
app/backend:1.0.0|company/backend:latest
app/frontend:1.0.0|company/frontend:latest
```

### Commandes

#### Export
```bash
# Exporter des images depuis un registry source
magina export -c config.brms --clean-on-error -v 1
```

#### Import
```bash
# Importer des images vers un registry destination
magina import -c config.brms --resume -v 2
```

#### Transfer
```bash
# Transfert complet entre registries
magina transfer -c config.brms --clean-on-error --resume -v 1
```

#### Validate
```bash
# Valider un fichier de configuration
magina validate -c config.brms
```

### Options Globales
- `-c, --config` : Fichier de configuration BRMS (obligatoire)
- `-v, --verbose` : Niveau de verbosité (0-3)
- `--version` : Affiche la version

### Options de Transfert
- `--clean-on-error` : Nettoie les images en cas d'erreur
- `--resume` : Continue l'opération même en cas d'erreur

## Développement

### Structure du Projet
```
magina/
├── cmd/            # Point d'entrée de l'application
├── internal/       # Code interne
├── docs/          # Documentation
├── .github/       # Configuration GitHub Actions
├── Taskfile.yml   # Tâches de build
└── README.md      # Documentation principale
```

### Commandes de Développement
```bash
# Installation des outils
task install

# Tests
task test

# Linting
task lint

# Build local
task build

# Build toutes plateformes
task release

# Générer la documentation CLI
task generate-docs
```

## Limitations Connues

- Pas de support des registries sans protocole
- Pas de parallélisation des opérations
- Pas de retry automatique en cas d'erreur réseau
- Pas de stockage persistant des credentials
- Pas de support natif IPv6-only

## Contribution

1. Fork le projet
2. Créer une branche (`git checkout -b feature/amazing-feature`)
3. Commit les changements (`git commit -m 'feat: add amazing feature'`)
4. Push la branche (`git push origin feature/amazing-feature`)
5. Ouvrir une Pull Request

## Licence

Ce projet est sous licence MIT. Voir le fichier [LICENSE](LICENSE) pour plus de détails.

## Support

Pour toute question ou problème :
1. Consulter les [Issues](https://github.com/Caezarr-OSS/magina/issues)
2. Ouvrir une nouvelle issue si nécessaire
3. Joindre les logs et la configuration (sans credentials)