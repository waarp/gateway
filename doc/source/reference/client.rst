Client invité de commande
#########################

.. program:: waarp-gateway

``waarp-gateway`` est l'application terminal permattant d'envoyer des commandes
à une instance waarp-gatewayd.



.. option:: status

   Affiche le statut de tous les services de la gateway interrogée.

   .. option:: --address ADDR, -a ADDR

      L'adresse de l'instance de gateway à interroger. Ce paramètre est requis.

   .. option:: --user USER, -u USER

      Le nom de l'utilisateur pour authentifier la requête. Ce paramètre est requis.

      Le mot de passe peut être renseigné avec la variable d'environnement
      ``WG_PASSWORD``. Si la variable d'environnement est vide, le mot de passe
      sera demandé dans l'invité de commande.