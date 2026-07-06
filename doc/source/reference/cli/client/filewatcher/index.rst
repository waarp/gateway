########################
Gestion des filewatchers
########################

Commande de gestion des *filewatchers*.

Un *filewatcher* permet de monitorer un dossier et de lancer des transferts
lorsque des fichiers y sont déposés.

Pour l'heure, seuls les dossiers distants sont supportés. Le *filewatcher*
remplace donc l'utilitaire ``get-remote`` livré avec Gateway, qui est donc
maintenant déprécié.

.. toctree::
   :maxdepth: 1

   add
   list
   get
   update
   delete
   start
   stop
   fire
