UPDATECONF
==========

Le traitement ``UPDATECONF`` extrait et importe un fichier de configuration
depuis une archive ZIP (typiquement générée via Waarp Manager). Si l'archive
contient également des fichiers de configuration pour les utilitaires
*filewatcher* et *get-remote*, ces fichiers seront copiés dans le répertoire
de configuration de l'instance Gateway.

Les fichiers contenus dans l'archive doivent impérativement avoir les noms suivants :

- ``nom-instance-gateway.json`` pour le fichier de configuration Gateway, où
  "nom-instance-gateway" est le nom de l'instance Gateway tel qu'il est déclaré
  dans le champ ``GatewayName`` du :doc:`fichier de configuration<../configuration>`
- ``fw.json`` pour le fichier de configuration du *filewatcher*
- ``get-file.list`` pour le fichier de configuration du *get-remote*

Les paramètres de la tâches sont :

* ``zipFile`` (*string*) - Optionnel. Le chemin de l'archive ZIP à extraire. Par
  défaut, le chemin du fichier de transfert sera utilisé.

