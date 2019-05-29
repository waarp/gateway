Client en ligne de commande
###########################

.. program:: waarp-gateway

``waarp-gateway`` est l'application terminal permettant d'envoyer des commandes
à une instance waarp-gatewayd.



.. option:: status

   Affiche le statut de tous les services de la gateway interrogée.

   .. option:: --address ADDR, -a ADDR

      L'adresse de l'instance de gateway à interroger. Ce paramètre est requis.

   .. option:: --user USER, -u USER

      Le nom de l'utilisateur pour authentifier la requête. Ce paramètre est requis.

   .. envvar::  WG_PASSWORD

      Le mot de passe de l'utilisateur. Si la variable d'environnement est vide,
      le mot de passe sera demandé dans l'invité de commande.