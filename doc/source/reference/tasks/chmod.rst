CHMOD
=====

Le traitement ``CHMOD`` permet de changer les permissions système sur le fichier
de transfert. Les paramètres de la tâche sont :

* ``perms`` (*number*) - Les permissions à appliquer au fichier au format octal
  (ex: ``644``). Les préfixes octales ``0`` et ``0o`` sont acceptés.
  Pour des raisons de sécurité, seuls les *bits* de permissions (``777``) sont
  autorisés pour cette tâche. Tous les autres (ex: ``1777`` ou ``2777``) seront
  refusés.

|

.. important::
   Cette tâche ne fonctionne que pour les fichiers se trouvant sur le système de
   fichier local. Si le fichier se trouve sur un stockage *cloud*, la tâche
   échouera.

.. attention::
   Windows traitant les permissions différemment des systèmes basés sur UNIX,
   seul le bit d'écriture pour le *owner* (soit le bit ``0o200``) est pris en
   compte. Tous les autres bits sont sans effet. Par convention, il est conseillé
   d'utiliser le mode ``0o400`` pour un fichier en lecture seule, et ``0o600``
   pour un fichier en lecture et écriture.