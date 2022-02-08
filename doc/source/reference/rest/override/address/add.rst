Ajouter une indirection
=======================

.. http:post:: /api/override/address

   Ajoute une ou plusieurs indirection(s) d'adresse.

   .. note::
      Si une adresse possède déjà une indirection, celle-ci sera écrasée
      par la nouvelle.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string <targetAddress>: Ajoute une indirection sur l'adresse
   <targetAddress> (la clé JSON). L'adresse définie en clé JSON est donc l'adresse
   remplacée, et l'adresse définie en valeur est l'adresse de remplacement. Il est
   possible de renseigner plusieurs paires d'adresses dans une même requête
   pour ajouter plusieurs indirections.

   :statuscode 201: La (les) indirection(s) ont été ajoutée(s) avec succès
   :statuscode 400: Un ou plusieurs des paramètres de la requête sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/override/address HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 64

         {
           "waarp.fr": "192.168.1.1",
           "waarp.org:6666": "localhost:8066"
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
