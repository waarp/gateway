Supprimer une indirection
=========================

.. http:delete:: /api/override/address/(string:address)

   Supprime l'indirection en place pour l'adresse donnée (si l'indirection existe).

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 204: L'indirection a été supprimée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: L'indirection demandée n'existe pas

   **Exemple de requête**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/override/address/waarp.fr HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 204 NO CONTENT
