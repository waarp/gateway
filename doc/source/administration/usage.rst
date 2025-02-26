Utilisation
===========

Lancement
---------

Une fois le fichier de configuration rempli, Waarp Gateway peut être lancée
avec la commande suivante :

.. code-block:: shell

   waarp-gatewayd -c chemin/de/la/configuration.ini


Arrêt
-----

Une fois lancée, Waarp Gateway peut être arrêtée en envoyant un signal
d'interruption. Il y a typiquement, 2 cas de figure :

- si Waarp Gateway a été lancée via un gestionnaire de service (ex:
  :program:`systemctl`), elle peut être arrêtée via ce même gestionnaire de
  service

- si Waarp Gateway a été lancée directement depuis un terminal, elle peut être
  arrêtée via la commande d'interruption (typiquement :kbd:`Control-C`) ou bien
  via un gestionnaire de tâches


Limitations des resources
-------------------------

Waarp Gateway permet à l'utilisateur de définir des limites sur les ressources
système utilisées par l'application via des variables d'environnement.

Ces variables d'environnement sont :

- :envvar:`WAARP_GATEWAYD_CPU_LIMIT`: La quantité maximale de processeurs logiques
  (ou cœurs) utilisés par l'application. La limite doit aussi être un nombre entier
  compris entre 1 et le nombre de processeurs logiques de la machine.
- :envvar:`WAARP_GATEWAYD_MEMORY_LIMIT`: La quantité de RAM maximale utilisée par
  l'application. À noter qu'il ne s'agit pas d'une limite *hard*. L'application
  tentera autant que possible de respecter cette limite, mais elle pourra parfois
  être amenée à la dépasser temporairement. La limite doit être un nombre (entier
  ou décimal), suivit d'une unité de taille (ex: ``2.5GB``). Les unités SI (``KB``,
  ``MB``, ``GB``...) et les unités IEC (``Kib``, ``Mib``, ``Gib``...) sont toutes
  deux acceptées. Les espaces ne sont pas significatifs, tout comme les lettres
  minuscules et majuscules.

Il est important de noter que fixer ces limites aura inévitablement un impact
sur les performances de l'application, en particulier si la charge de transferts
sur l'application est importante. Il est donc déconseiller de fixer des limites
de ressources si cela n'est pas absolument nécessaire.
