Mettre un transfert en pause
============================

.. http:put:: /api/transfers/(int:transfer_id)/pause

   Pause le transfert portant l'identifiant ``transfer_id``. Le transfert
   doit être en cours ou plannifié.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          PUT https://my_waarp_gateway.net/api/transfers/1/pause HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 201: Le transfert a été mis en pause avec succès
   :statuscode 400: Le transfert demandé ne peut pas être mis en pause
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le transfert demandé n'existe pas

   :resheader Location: Le chemin d'accès au transfert redémmaré

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/transfers/1
