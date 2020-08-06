Utilisation
===========

Lancement
---------

   Une fois le fichier de configuration rempli, la *gateway* peut être lancée
   avec la commande suivante :

   .. code-block:: shell

      waarp-gatewayd -c chemin/de/la/configuration.ini


Arrêt
-----

   Une fois lancée, la gateway peut être arrêtée en envoyant un signal
   d'interruption. Il y a typiquement, 2 cas de figure :

   - si la gateway a été lancée via un gestionnaire de service (ex: `systemd`),
     elle peut être arrêtée via ce même gestionnaire de service

   - si la gateway a été lancée directement depuis un terminal, elle peut être
     arrêtée via la commande d'interruption (typiquement ``Ctrl+C``) ou bien
     via un gestionnaire de tâches