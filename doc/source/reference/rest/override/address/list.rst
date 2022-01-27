Lister les indirections
=======================

.. http:get:: /api/override/address

   Renvoi une liste des indirections d'adresse existantes.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de la requête sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson string <targetAddress>: Ajoute une indirection sur l'adresse
      <targetAddress> (la clé JSON). L'adresse définie en clé JSON est donc l'adresse
      remplacée, et l'adresse définie en valeur est l'adresse de remplacement.
      Chaque paire clé-valeur correspond à une indirection.

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/override/address HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 64

         {
           "waarp.fr": "192.168.1.1",
           "waarp.org:6666": "localhost:8066"
         }