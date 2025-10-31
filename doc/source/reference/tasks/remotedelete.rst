REMOTEDELETE
============

Le traitement ``REMOTEDELETE`` supprime un fichier ou dossier sur le partenaire
distant du transfert en cours.

.. note::
   Cette tâche ne fonctionnera que si le protocole de transfert permet la suppression
   à distance. Par conséquent, cette tâche ne marchera que si Gateway est client
   du transfert, et uniquement avec les protocoles *SFTP*, *FTP* et *HTTP*.

   À noter également que pour que la commande fonctionne, le partenaire doit
   autoriser la suppression à distance. Cette tâche ne marchera donc pas si le
   partenaire distant est une autre Gateway.

* ``path`` (*string*) - Le chemin du fichier/dossier à supprimer. Par défaut,
  le fichier de transfert sera supprimé.
* ``recursive`` (*bool*) - Indique si Gateway doit d'abord récursivement supprimer
  le contenu du dossier distant avant de le supprimer. Par défaut, la suppression
  récursive n'est pas activée. Cette option n'a pas d'effet si le chemin pointe
  sur un fichier.

  Dans la plupart des cas, il est impossible de supprimer un dossier non-vide.
  Avec cette option active, Gateway supprimera récursivement tout le contenu
  du dossier cible avant de le supprimer.

  *Note:* Le protocole doit permettre de lister le contenu d'un fichier à distance
  pour pouvoir utiliser cette option. HTTP n'est donc pas supporté.
* ``timeout`` (*string*) - La durée limite de la requête. Si la commande dure
  plus longtemps que cette durée, la tâche sera annulée et tombera en erreur.
  Les unités de temps acceptées sont : ``s`` (secondes), ``m`` (minutes),
  et ``h`` (heures).
  Par défaut, la tâche n'a pas de durée limite.