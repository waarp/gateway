.. _reference-cli-client-cloud:

###########################
Gestion des instances cloud
###########################

Les instances cloud représentent une alternative au stockage local des fichiers
de transfert. Les fichiers envoyés ou reçus par la Gateway peuvent être stockés
sur une instance de stockage cloud au lieu du disque local de la machine sur
laquelle est installée la Gateway.

Une fois l'instance cloud configurée, il est possible de l'utiliser pour un
transfert en remplaçant le chemin du fichier par un URL sous la forme:
``type_instance://nom_instance/chemin/du/fichier``

À noter que ajouter/modifier/supprimer les instances cloud requiert les droits
d'administration sur la Gateway.

.. toctree::
   :maxdepth: 1

   add
   list
   get
   update
   delete
